package services

import (
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/leandro-lugaresi/hub"
	"github.com/muety/artifex/v2"
	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/repositories"
	"github.com/muety/wakapi/utils"
	"github.com/patrickmn/go-cache"
)

type LeaderboardService struct {
	config         *config.Config
	cache          *cache.Cache
	eventBus       *hub.Hub
	repository     repositories.ILeaderboardRepository
	summaryService ISummaryService
	userService    IUserService
	teamService    ITeamService
	queueDefault   *artifex.Dispatcher
	queueWorkers   *artifex.Dispatcher
	defaultScope   *models.IntervalKey
}

func NewLeaderboardService(leaderboardRepo repositories.ILeaderboardRepository, summaryService ISummaryService, userService IUserService, teamService ITeamService) *LeaderboardService {
	srv := &LeaderboardService{
		config:         config.Get(),
		cache:          cache.New(6*time.Hour, 6*time.Hour),
		eventBus:       config.EventBus(),
		repository:     leaderboardRepo,
		summaryService: summaryService,
		userService:    userService,
		teamService:    teamService,
		queueDefault:   config.GetDefaultQueue(),
		queueWorkers:   config.GetQueue(config.QueueProcessing),
	}

	scope, err := helpers.ParseInterval(srv.config.App.LeaderboardScope)
	if err != nil {
		config.Log().Fatal(err.Error())
	}
	srv.defaultScope = scope

	onUserUpdate := srv.eventBus.Subscribe(0, config.EventUserUpdate)
	go func(sub *hub.Subscription) {
		for m := range sub.Receiver {

			// regenerate leaderboard for updated user, if leaderboard enabled and none present, yet
			user := m.Fields[config.FieldPayload].(*models.User)

			exists, err := srv.ExistsAnyByUser(user.ID)
			if err != nil {
				config.Log().Error("failed to check existing leaderboards upon user update", "error", err)
			}

			if user.PublicLeaderboard && !exists {
				slog.Info("generating leaderboard after settings update", "userID", user.ID)
				srv.ComputeLeaderboard([]*models.User{user}, srv.defaultScope, []uint8{models.SummaryLanguage})
				srv.ComputeTeamLeaderboard(srv.defaultScope)
			} else if !user.PublicLeaderboard && exists {
				slog.Info("clearing leaderboard after settings update", "userID", user.ID)
				if err := srv.repository.DeleteByUser(user.ID); err != nil {
					config.Log().Error("failed to clear leaderboard for user", "userID", user.ID, "error", err)
				}
				srv.ComputeTeamLeaderboard(srv.defaultScope)
			}
		}
	}(&onUserUpdate)

	return srv
}

func (srv *LeaderboardService) GetDefaultScope() *models.IntervalKey {
	return srv.defaultScope
}

func (srv *LeaderboardService) Schedule() {
	slog.Info("scheduling leaderboard generation")

	generate := func() {
		users, err := srv.userService.GetAllByLeaderboard(true)
		if err != nil {
			config.Log().Error("failed to get users for leaderboard generation", "error", err)
			return
		}
		srv.ComputeLeaderboard(users, srv.defaultScope, []uint8{models.SummaryLanguage})
		srv.ComputeTeamLeaderboard(srv.defaultScope)
	}

	for _, cronExp := range srv.config.App.GetLeaderboardGenerationTimeCron() {
		if _, err := srv.queueDefault.DispatchCron(generate, cronExp); err != nil {
			config.Log().Error("failed to schedule leaderboard generation", "cronExpression", cronExp, "error", err)
		}
	}
}

func (srv *LeaderboardService) ComputeLeaderboard(users []*models.User, interval *models.IntervalKey, by []uint8) error {
	slog.Info("generating leaderboard", "interval", (*interval)[0], "userCount", len(users), "aggregationCount", len(by))

	for _, user := range users {
		if err := srv.repository.DeleteByUserAndInterval(user.ID, interval); err != nil {
			config.Log().Error("failed to delete leaderboard items for user", "userID", user.ID, "interval", (*interval)[0], "error", err)
			continue
		}

		item, err := srv.GenerateByUser(user, interval)
		if err != nil {
			config.Log().Error("failed to regenerate general leaderboard for user", "userID", user.ID, "error", err)
			continue
		}

		if err := srv.repository.InsertBatch([]*models.LeaderboardItem{item}); err != nil {
			config.Log().Error("failed to persist general leaderboard for user", "userID", user.ID, "error", err)
			continue
		}

		for _, by := range by {
			items, err := srv.GenerateAggregatedByUser(user, interval, by)
			if err != nil {
				config.Log().Error("failed to regenerate aggregated leaderboard for user", "aggregatedBy", models.GetEntityColumn(by), "userID", user.ID, "error", err)
				continue
			}

			if len(items) == 0 {
				continue
			}

			if err := srv.repository.InsertBatch(items); err != nil {
				config.Log().Error("failed to persist aggregated leaderboard for user", "aggregatedBy", models.GetEntityColumn(by), "userID", user.ID, "error", err)
				continue
			}
		}
	}

	srv.cache.Flush()
	slog.Info("finished leaderboard generation")
	return nil
}

func (srv *LeaderboardService) ExistsAnyByUser(userId string) (bool, error) {
	count, err := srv.repository.CountAllByUser(userId)
	return count > 0, err
}

func (srv *LeaderboardService) CountUsers(excludeZero bool) (int64, error) {
	// check cache
	cacheKey := fmt.Sprintf("count_total_%v", excludeZero)
	if cacheResult, ok := srv.cache.Get(cacheKey); ok {
		return cacheResult.(int64), nil
	}

	count, err := srv.repository.CountUsers(excludeZero)
	if err != nil {
		srv.cache.SetDefault(cacheKey, count)
	}
	return count, err
}

func (srv *LeaderboardService) GetByInterval(interval *models.IntervalKey, pageParams *utils.PageParams, resolveUsers bool) (models.Leaderboard, error) {
	return srv.GetAggregatedByInterval(interval, nil, pageParams, resolveUsers)
}

func (srv *LeaderboardService) GetByIntervalAndUser(interval *models.IntervalKey, userId string, resolveUser bool) (models.Leaderboard, error) {
	return srv.GetAggregatedByIntervalAndUser(interval, userId, nil, resolveUser)
}

func (srv *LeaderboardService) GetAggregatedByInterval(interval *models.IntervalKey, by *uint8, pageParams *utils.PageParams, resolveUsers bool) (models.Leaderboard, error) {
	// check cache
	cacheKey := srv.getHash(interval, by, "", pageParams)
	if cacheResult, ok := srv.cache.Get(cacheKey); ok {
		return cacheResult.([]*models.LeaderboardItemRanked), nil
	}

	items, err := srv.repository.GetAllAggregatedByInterval(interval, by, pageParams.Limit(), pageParams.Offset())
	if err != nil {
		return nil, err
	}

	if resolveUsers {
		users, err := srv.userService.GetManyMapped(models.Leaderboard(items).UserIDs())
		if err != nil {
			config.Log().Error("failed to resolve users for leaderboard item", "error", err)
		} else {
			for _, item := range items {
				if u, ok := users[item.UserID]; ok {
					item.User = u
				}
			}
		}
	}

	srv.cache.SetDefault(cacheKey, items)
	return items, nil
}

func (srv *LeaderboardService) GetAggregatedByIntervalAndUser(interval *models.IntervalKey, userId string, by *uint8, resolveUser bool) (models.Leaderboard, error) {
	// check cache
	cacheKey := srv.getHash(interval, by, userId, nil)
	if cacheResult, ok := srv.cache.Get(cacheKey); ok {
		return cacheResult.([]*models.LeaderboardItemRanked), nil
	}

	items, err := srv.repository.GetAggregatedByUserAndInterval(userId, interval, by, 0, 0)
	if err != nil {
		return nil, err
	}

	if resolveUser {
		u, err := srv.userService.GetUserById(userId)
		if err != nil {
			config.Log().Error("failed to resolve user for leaderboard item", "error", err)
		} else {
			for _, item := range items {
				item.User = u
			}
		}
	}

	srv.cache.SetDefault(cacheKey, items)
	return items, nil
}

func (srv *LeaderboardService) GenerateByUser(user *models.User, interval *models.IntervalKey) (*models.LeaderboardItem, error) {
	err, from, to := helpers.ResolveIntervalTZ(interval, user.TZ(), user.StartOfWeekDay())
	if err != nil {
		return nil, err
	}

	timeout := models.DefaultHeartbeatsTimeout
	summary, err := srv.summaryService.Aliased(from, to, user, srv.summaryService.Retrieve, nil, &timeout, false)
	if err != nil {
		return nil, err
	}

	// exclude unknown language (will also exclude browsing time by chrome-wakatime plugin)
	total := summary.TotalTime() - summary.TotalTimeByKey(models.SummaryLanguage, models.UnknownSummaryKey)
	return &models.LeaderboardItem{
		User:     user,
		UserID:   user.ID,
		Interval: (*interval)[0],
		Total:    total,
	}, nil
}

func (srv *LeaderboardService) GenerateAggregatedByUser(user *models.User, interval *models.IntervalKey, by uint8) ([]*models.LeaderboardItem, error) {
	err, from, to := helpers.ResolveIntervalTZ(interval, user.TZ(), user.StartOfWeekDay())
	if err != nil {
		return nil, err
	}

	summary, err := srv.summaryService.Aliased(from, to, user, srv.summaryService.Retrieve, nil, nil, false)
	if err != nil {
		return nil, err
	}

	summaryItems := *summary.GetByType(by)
	items := make([]*models.LeaderboardItem, 0, summaryItems.Len())

	for _, item := range summaryItems {
		// explicitly exclude unknown languages from leaderboard
		if item.Key == models.UnknownSummaryKey {
			continue
		}

		items = append(items, &models.LeaderboardItem{
			User:     user,
			UserID:   user.ID,
			Interval: (*interval)[0],
			By:       &by,
			Total:    summary.TotalTimeByKey(by, item.Key),
			Key:      &item.Key,
		})
	}

	return items, nil
}

func (srv *LeaderboardService) ComputeTeamLeaderboard(interval *models.IntervalKey) error {
	slog.Info("generating team leaderboard", "interval", (*interval)[0])

	if err := srv.repository.DeleteTeamByInterval(interval); err != nil {
		return fmt.Errorf("failed to delete old team leaderboard items: %w", err)
	}

	teams, err := srv.teamService.GetAll()
	if err != nil {
		return fmt.Errorf("failed to get teams: %w", err)
	}

	items := make([]*models.TeamLeaderboardItem, 0, len(teams))
	for _, team := range teams {
		members, err := srv.teamService.GetMembers(team.ID)
		if err != nil {
			config.Log().Error("failed to get team members for leaderboard", "error", err, "team", team.ID)
			continue
		}

		memberIDs := make([]string, len(members))
		for i, m := range members {
			memberIDs[i] = m.UserID
		}

		if len(memberIDs) == 0 {
			continue
		}

		total, err := srv.repository.SumByUsersAndInterval(memberIDs, interval)
		if err != nil {
			config.Log().Error("failed to sum leaderboard for team", "error", err, "team", team.ID)
			continue
		}

		topLangs, err := srv.repository.TopKeysByUsersAndInterval(memberIDs, interval, models.SummaryLanguage, 3)
		if err != nil {
			config.Log().Error("failed to get top languages for team", "error", err, "team", team.ID)
		}

		item := &models.TeamLeaderboardItem{
			TeamID:       team.ID,
			TeamName:     team.Name,
			Interval:     (*interval)[0],
			MemberCount:  len(members),
			Total:        total,
			TopLanguages: topLangs,
		}
		items = append(items, item)
	}

	if len(items) > 0 {
		if err := srv.repository.InsertTeamBatch(items); err != nil {
			return fmt.Errorf("failed to insert team leaderboard items: %w", err)
		}
	}

	srv.cache.Flush()
	slog.Info("finished team leaderboard generation", "teamCount", len(items))
	return nil
}

func (srv *LeaderboardService) GetTeamLeaderboard(interval *models.IntervalKey) (models.TeamLeaderboard, error) {
	cacheKey := "team_leaderboard__" + strings.Join(*interval, "__")
	if cacheResult, ok := srv.cache.Get(cacheKey); ok {
		return cacheResult.(models.TeamLeaderboard), nil
	}

	dbItems, err := srv.repository.GetTeamLeaderboardByInterval(interval)
	if err != nil {
		return nil, err
	}

	items := make(models.TeamLeaderboard, len(dbItems))
	for i, dbItem := range dbItems {
		items[i] = &models.TeamLeaderboardItemRanked{
			TeamLeaderboardItem: *dbItem,
			Rank:                uint(i + 1),
		}
	}

	srv.cache.SetDefault(cacheKey, items)
	return items, nil
}

func (srv *LeaderboardService) getHash(interval *models.IntervalKey, by *uint8, user string, pageParams *utils.PageParams) string {
	k := strings.Join(*interval, "__") + "__" + user
	if by != nil && !reflect.ValueOf(by).IsNil() {
		k += "__" + models.GetEntityColumn(*by)
	}
	if pageParams != nil {
		k += "__" + strconv.Itoa(pageParams.Page) + "__" + strconv.Itoa(pageParams.PageSize)
	}
	return k
}
