package routes

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/go-chi/chi/v5"
	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/middlewares"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/models/view"
	routeutils "github.com/muety/wakapi/routes/utils"
	"github.com/muety/wakapi/services"
	"github.com/muety/wakapi/utils"
	i18n "github.com/muety/wakapi/views/i18n"
)

const (
	memberSummaryMinRangeDays = 3
	memberSummaryMaxRangeDays = 31
)

type TeamsHandler struct {
	config         *conf.Config
	userSrvc       services.IUserService
	teamSrvc       services.ITeamService
	summarySrvc    services.ISummaryService
	heartbeatsSrvc services.IHeartbeatService
	durationSrvc   services.IDurationService
	aliasSrvc      services.IAliasService
}

func NewTeamsHandler(
	userService services.IUserService,
	teamService services.ITeamService,
	summaryService services.ISummaryService,
	heartbeatService services.IHeartbeatService,
	durationService services.IDurationService,
	aliasService services.IAliasService,
) *TeamsHandler {
	return &TeamsHandler{
		config:         conf.Get(),
		userSrvc:       userService,
		teamSrvc:       teamService,
		summarySrvc:    summaryService,
		heartbeatsSrvc: heartbeatService,
		durationSrvc:   durationService,
		aliasSrvc:      aliasService,
	}
}

func (h *TeamsHandler) RegisterRoutes(router chi.Router) {
	r := chi.NewRouter()

	authMiddleware := middlewares.NewAuthenticateMiddleware(h.userSrvc)
	authMiddleware = authMiddleware.WithRedirectTarget(defaultErrorRedirectTarget())
	authMiddleware = authMiddleware.WithRedirectErrorMessage("unauthorized")

	r.Use(authMiddleware.Handler)
	r.Get("/", h.GetIndex)
	r.Get("/{id}", h.GetTeamDetail)
	r.Get("/{id}/members/{userID}", h.GetMemberSummary)

	router.Mount("/teams", r)
}

func (h *TeamsHandler) GetIndex(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}

	user := middlewares.GetPrincipal(r)

	var teams []*models.Team
	var err error
	if user.IsAdmin {
		teams, err = h.teamSrvc.GetAll()
	} else {
		teams, err = h.teamSrvc.GetByUser(user.ID)
	}
	if err != nil {
		teams = []*models.Team{}
	}

	teamMemberCounts := make(map[string]int64)
	for _, team := range teams {
		if count, err := h.teamSrvc.CountMembers(team.ID); err == nil {
			teamMemberCounts[team.ID] = count
		}
	}

	vm := &view.TeamsViewModel{
		SharedLoggedInViewModel: view.SharedLoggedInViewModel{
			SharedViewModel: view.NewSharedViewModel(h.config, nil, r, user),
			User:            user,
		},
		Teams:            teams,
		TeamMemberCounts: teamMemberCounts,
	}

	if err := templates[conf.TeamsTemplate].Execute(w, routeutils.WithSessionMessages(vm, r, w)); err != nil {
		conf.Log().Request(r).Error("failed to render teams page", "error", err)
	}
}

func (h *TeamsHandler) GetTeamDetail(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}

	user := middlewares.GetPrincipal(r)
	teamID := strings.TrimSpace(chi.URLParam(r, "id"))

	if teamID == "" {
		http.Redirect(w, r, fmt.Sprintf("%s/teams", h.config.Server.BasePath), http.StatusFound)
		return
	}

	lang := routeutils.ResolveLanguage(r, user)

	team, err := h.teamSrvc.GetByID(teamID)
	if err != nil {
		routeutils.SetError(r, w, i18n.Translate(lang, "flash.team_not_found"))
		http.Redirect(w, r, fmt.Sprintf("%s/teams", h.config.Server.BasePath), http.StatusFound)
		return
	}

	if !user.IsAdmin {
		isMember, err := h.teamSrvc.IsTeamMember(teamID, user.ID)
		if err != nil {
			conf.Log().Request(r).Error("failed to check team membership", "error", err, "team", teamID, "user", user.ID)
			routeutils.SetError(r, w, i18n.Translate(lang, "flash.internal_error"))
			http.Redirect(w, r, fmt.Sprintf("%s/teams", h.config.Server.BasePath), http.StatusFound)
			return
		}
		if !isMember {
			routeutils.SetError(r, w, i18n.Translate(lang, "flash.not_your_team"))
			http.Redirect(w, r, fmt.Sprintf("%s/teams", h.config.Server.BasePath), http.StatusFound)
			return
		}
	}

	members, _ := h.teamSrvc.GetMembers(teamID)

	to := time.Now()
	from := to.AddDate(0, 0, -7)

	memberSummaries := make([]*models.Summary, 0, len(members))
	memberTotals := make([]*view.MemberSummaryItem, 0, len(members))

	for _, member := range members {
		memberUser, err := h.userSrvc.GetUserById(member.UserID)
		if err != nil {
			conf.Log().Request(r).Error("failed to get team member user",
				"error", err, "user", member.UserID, "team", teamID)
			continue
		}
		summary, err := h.summarySrvc.Aliased(
			from, to, memberUser,
			h.summarySrvc.Retrieve,
			&models.Filters{}, nil, false,
		)
		if err != nil {
			conf.Log().Request(r).Error("failed to fetch summary for team member",
				"error", err, "user", member.UserID, "team", teamID)
			continue
		}
		memberSummaries = append(memberSummaries, summary)
		memberTotals = append(memberTotals, &view.MemberSummaryItem{
			UserID:    member.UserID,
			TotalTime: summary.TotalTime(),
		})
	}

	aggregated, err := h.summarySrvc.MergeSummariesAcrossUsers(memberSummaries)
	if err != nil {
		conf.Log().Request(r).Warn("failed to merge member summaries", "error", err, "team", teamID)
		aggregated = models.NewEmptySummary()
	}
	aggregated = aggregated.Sorted()

	// Limit to top 5 per category
	if len(aggregated.Projects) > 5 {
		aggregated.Projects = aggregated.Projects[:5]
	}
	if len(aggregated.Languages) > 5 {
		aggregated.Languages = aggregated.Languages[:5]
	}
	if len(aggregated.Editors) > 5 {
		aggregated.Editors = aggregated.Editors[:5]
	}
	if len(aggregated.OperatingSystems) > 5 {
		aggregated.OperatingSystems = aggregated.OperatingSystems[:5]
	}

	isOwner, _ := h.teamSrvc.IsTeamOwner(teamID, user.ID)

	vm := &view.TeamDetailViewModel{
		SharedLoggedInViewModel: view.SharedLoggedInViewModel{
			SharedViewModel: view.NewSharedViewModel(h.config, nil, r, user),
			User:            user,
		},
		Team:            team,
		Members:         members,
		Summary:         aggregated,
		MemberSummaries: memberTotals,
		From:            from,
		To:              to,
		IntervalLabel:   "Last 7 days",
		IsOwner:         isOwner,
	}

	if err := templates[conf.TeamDetailTemplate].Execute(w, routeutils.WithSessionMessages(vm, r, w)); err != nil {
		conf.Log().Request(r).Error("failed to render team detail page", "error", err)
	}
}

func (h *TeamsHandler) GetMemberSummary(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}

	user := middlewares.GetPrincipal(r)
	teamID := strings.TrimSpace(chi.URLParam(r, "id"))
	memberUserID := strings.TrimSpace(chi.URLParam(r, "userID"))

	if teamID == "" || memberUserID == "" {
		http.Redirect(w, r, fmt.Sprintf("%s/teams", h.config.Server.BasePath), http.StatusFound)
		return
	}

	lang := routeutils.ResolveLanguage(r, user)

	// Verify team exists
	team, err := h.teamSrvc.GetByID(teamID)
	if err != nil {
		routeutils.SetError(r, w, i18n.Translate(lang, "flash.team_not_found"))
		http.Redirect(w, r, fmt.Sprintf("%s/teams", h.config.Server.BasePath), http.StatusFound)
		return
	}

	// Authorization: must be admin or team owner
	isOwner, _ := h.teamSrvc.IsTeamOwner(teamID, user.ID)
	if !user.IsAdmin && !isOwner {
		routeutils.SetError(r, w, i18n.Translate(lang, "flash.unauthorized_member_dashboard"))
		http.Redirect(w, r, fmt.Sprintf("%s/teams/%s", h.config.Server.BasePath, teamID), http.StatusFound)
		return
	}

	// Verify target user is a team member
	isMember, err := h.teamSrvc.IsTeamMember(teamID, memberUserID)
	if err != nil {
		conf.Log().Request(r).Error("failed to check team membership", "error", err, "team", teamID, "user", memberUserID)
		routeutils.SetError(r, w, i18n.Translate(lang, "flash.internal_error"))
		http.Redirect(w, r, fmt.Sprintf("%s/teams/%s", h.config.Server.BasePath, teamID), http.StatusFound)
		return
	}
	if !isMember {
		routeutils.SetError(r, w, i18n.Translate(lang, "flash.not_team_member"))
		http.Redirect(w, r, fmt.Sprintf("%s/teams/%s", h.config.Server.BasePath, teamID), http.StatusFound)
		return
	}

	// Get the target member user
	memberUser, err := h.userSrvc.GetUserById(memberUserID)
	if err != nil {
		conf.Log().Request(r).Error("failed to get team member user", "error", err, "user", memberUserID)
		routeutils.SetError(r, w, i18n.Translate(lang, "flash.user_not_found"))
		http.Redirect(w, r, fmt.Sprintf("%s/teams/%s", h.config.Server.BasePath, teamID), http.StatusFound)
		return
	}

	// Default to "today" if no interval specified (skip cookie redirect)
	q := r.URL.Query()
	if q.Get("interval") == "" && q.Get("from") == "" {
		q.Set("interval", "today")
		r.URL.RawQuery = q.Encode()
	}

	rawQuery := r.URL.RawQuery

	// Parse summary params for the target member
	summaryParams, err := helpers.ParseSummaryParamsForUser(r, memberUser)
	if err != nil {
		conf.Log().Request(r).Error("failed to parse summary params", "error", err, "user", memberUserID)
		w.WriteHeader(http.StatusBadRequest)
		templates[conf.SummaryTemplate].Execute(w, h.buildMemberSummaryViewModel(r, w, user, team, memberUserID).WithError("invalid date range"))
		return
	}

	summary, err, status := routeutils.LoadUserSummaryByParams(h.summarySrvc, summaryParams)
	if err != nil {
		conf.Log().Request(r).Error("failed to load member summary", "error", err, "user", memberUserID)
		w.WriteHeader(status)
		templates[conf.SummaryTemplate].Execute(w, h.buildMemberSummaryViewModel(r, w, user, team, memberUserID).WithError(err.Error()))
		return
	}

	summaryWithoutFilter, err, status := routeutils.LoadUserSummaryWithoutFilter(h.summarySrvc, summaryParams)
	if err != nil {
		conf.Log().Request(r).Error("failed to load member summary", "error", err, "user", memberUserID)
		w.WriteHeader(status)
		templates[conf.SummaryTemplate].Execute(w, h.buildMemberSummaryViewModel(r, w, user, team, memberUserID).WithError(err.Error()))
		return
	}
	availableFilters := h.extractAvailableFilters(summaryWithoutFilter)

	// User first data
	firstData, err := h.heartbeatsSrvc.GetFirstByUser(memberUser)
	if err != nil {
		conf.Log().Request(r).Error("error fetching member's heartbeats range", "user", memberUserID, "error", err)
	}

	// Timeline data (daily stats)
	var timeline []*view.TimelineViewModel
	if rangeDays := summaryParams.RangeDays(); rangeDays >= memberSummaryMinRangeDays && rangeDays <= memberSummaryMaxRangeDays {
		dailyStatsSummaries, err := h.fetchSplitSummaries(summaryParams)
		if err != nil {
			conf.Log().Request(r).Error("failed to load timeline stats", "error", err)
		} else {
			timeline = view.NewTimelineViewModel(dailyStatsSummaries)
		}
	}

	// Hourly breakdown data
	var hourlyBreakdown view.HourlyBreakdownsViewModel
	hourlyBreakdownFrom := summaryParams.From
	if summaryParams.RangeDays() > 1 {
		hourlyBreakdownFrom = summaryParams.To.Add(-24 * time.Hour)
	}
	if durations, err := h.durationSrvc.Get(hourlyBreakdownFrom, summaryParams.To, summaryParams.User, summaryParams.Filters, nil, false); err == nil {
		if len(durations) <= 200 {
			hourlyBreakdown = view.NewHourlyBreakdownViewModel(view.NewHourlyBreakdownItems(durations, func(t uint8, k string) string {
				s, _ := h.aliasSrvc.GetAliasOrDefault(memberUser.ID, t, k)
				return s
			}))
		}
	} else {
		conf.Log().Request(r).Error("failed to load hourly breakdown stats", "error", err)
	}

	vm := view.SummaryViewModel{
		SharedLoggedInViewModel: view.SharedLoggedInViewModel{
			SharedViewModel: view.NewSharedViewModel(h.config, nil, r, user),
			User:            user,
		},
		Summary:             summary,
		SummaryParams:       summaryParams,
		AvailableFilters:    availableFilters,
		EditorColors:        routeutils.FilterColors(h.config.App.GetEditorColors(), summary.Editors),
		LanguageColors:      routeutils.FilterColors(h.config.App.GetLanguageColors(), summary.Languages),
		OSColors:            routeutils.FilterColors(h.config.App.GetOSColors(), summary.OperatingSystems),
		RawQuery:            rawQuery,
		UserFirstData:       firstData,
		DataRetentionMonths: h.config.App.DataRetentionMonths,
		Timeline:            timeline,
		HourlyBreakdown:     hourlyBreakdown,
		HourlyBreakdownFrom: hourlyBreakdownFrom,
		TeamContext: &view.TeamMemberViewContext{
			TeamID:   teamID,
			TeamName: team.Name,
			MemberID: memberUserID,
		},
	}

	templates[conf.SummaryTemplate].Execute(w, vm)
}

func (h *TeamsHandler) buildMemberSummaryViewModel(r *http.Request, w http.ResponseWriter, user *models.User, team *models.Team, memberID string) *view.SummaryViewModel {
	return routeutils.WithSessionMessages(&view.SummaryViewModel{
		SharedLoggedInViewModel: view.SharedLoggedInViewModel{
			User:            user,
			SharedViewModel: view.NewSharedViewModel(h.config, nil, r, user),
		},
		TeamContext: &view.TeamMemberViewContext{
			TeamID:   team.ID,
			TeamName: team.Name,
			MemberID: memberID,
		},
	}, r, w)
}

func (h *TeamsHandler) fetchSplitSummaries(params *models.SummaryParams) ([]*models.Summary, error) {
	summaries := make([]*models.Summary, 0)
	intervals := utils.SplitRangeByDays(params.From, params.To)
	for _, interval := range intervals {
		curSummary, err := h.summarySrvc.Aliased(interval[0], interval[1], params.User, h.summarySrvc.Retrieve, params.Filters, nil, false)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, curSummary)
	}
	return summaries, nil
}

func (h *TeamsHandler) extractAvailableFilters(summary *models.Summary) view.AvailableFilters {
	return view.AvailableFilters{
		ProjectNames:  slice.Map(summary.Projects, func(_ int, item *models.SummaryItem) string { return item.Key }),
		LanguageNames: slice.Map(summary.Languages, func(_ int, item *models.SummaryItem) string { return item.Key }),
		MachineNames:  slice.Map(summary.Machines, func(_ int, item *models.SummaryItem) string { return item.Key }),
		LabelNames:    slice.Map(summary.Labels, func(_ int, item *models.SummaryItem) string { return item.Key }),
		CategoryNames: slice.Map(summary.Categories, func(_ int, item *models.SummaryItem) string { return item.Key }),
	}
}
