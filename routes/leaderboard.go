package routes

import (
	"fmt"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/go-chi/chi/v5"
	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/middlewares"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/models/view"
	routeutils "github.com/muety/wakapi/routes/utils"
	"github.com/muety/wakapi/services"
	"github.com/muety/wakapi/utils"
	"net/http"
	"strings"
)

type LeaderboardHandler struct {
	config             *conf.Config
	userService        services.IUserService
	leaderboardService services.ILeaderboardService
	teamService        services.ITeamService
}

var allowedAggregations = map[string]uint8{
	"language": models.SummaryLanguage,
}

func NewLeaderboardHandler(userService services.IUserService, leaderboardService services.ILeaderboardService, teamService services.ITeamService) *LeaderboardHandler {
	return &LeaderboardHandler{
		config:             conf.Get(),
		userService:        userService,
		leaderboardService: leaderboardService,
		teamService:        teamService,
	}
}

func (h *LeaderboardHandler) RegisterRoutes(router chi.Router) {
	r := chi.NewRouter()

	authMiddleware := middlewares.NewAuthenticateMiddleware(h.userService)
	authMiddleware = authMiddleware.WithRedirectTarget(defaultErrorRedirectTarget())
	authMiddleware = authMiddleware.WithRedirectErrorMessage("unauthorized")

	r.Use(authMiddleware.Handler)
	r.Get("/", h.GetIndex)

	router.Mount("/leaderboard", r)
}

func (h *LeaderboardHandler) GetIndex(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}

	if err := templates[conf.LeaderboardTemplate].Execute(w, h.buildViewModel(r, w)); err != nil {
		conf.Log().Request(r).Error("failed to get leaderboard page", "error", err)
	}
}

func (h *LeaderboardHandler) buildViewModel(r *http.Request, w http.ResponseWriter) *view.LeaderboardViewModel {
	user := middlewares.GetPrincipal(r)
	tabParam := strings.ToLower(r.URL.Query().Get("tab"))
	byParam := strings.ToLower(r.URL.Query().Get("by"))
	keyParam := strings.ToLower(r.URL.Query().Get("key"))
	pageParams := utils.ParsePageParamsWithDefault(r, 1, 100)
	// note: pagination is not fully implemented, yet
	// count function to get total item / total pages is missing
	// and according ui (+ optionally search bar) is missing, too

	// Determine active tab
	tab := "total"
	if tabParam == "teams" {
		tab = "teams"
	} else if byParam != "" {
		tab = "language"
	}

	if tab == "teams" {
		teamItems, err := h.leaderboardService.GetTeamLeaderboard(h.leaderboardService.GetDefaultScope())
		if err != nil {
			conf.Log().Request(r).Error("error while fetching team leaderboard", "error", err)
			return &view.LeaderboardViewModel{
				SharedLoggedInViewModel: view.SharedLoggedInViewModel{
					SharedViewModel: view.NewSharedViewModel(h.config, &view.Messages{Error: criticalError}, r, user),
					User:            user,
				},
				Tab: tab,
			}
		}

		userTeamIDs := make(map[string]bool)
		if user != nil {
			if teams, err := h.teamService.GetByUser(user.ID); err == nil {
				for _, t := range teams {
					userTeamIDs[t.ID] = true
				}
			}
		}

		vm := &view.LeaderboardViewModel{
			SharedLoggedInViewModel: view.SharedLoggedInViewModel{
				SharedViewModel: view.NewSharedViewModel(h.config, nil, r, user),
				User:            user,
			},
			Tab:           tab,
			TeamItems:     teamItems,
			UserTeamIDs:   userTeamIDs,
			IntervalLabel: h.leaderboardService.GetDefaultScope().GetHumanReadable(),
			PageParams:    pageParams,
		}
		return routeutils.WithSessionMessages(vm, r, w)
	}

	var err error
	var leaderboard models.Leaderboard
	var userLanguages map[string][]string
	var topKeys []string

	if byParam == "" {
		leaderboard, err = h.leaderboardService.GetByInterval(h.leaderboardService.GetDefaultScope(), pageParams, true)
		if err != nil {
			conf.Log().Request(r).Error("error while fetching general leaderboard items", "error", err)
			return &view.LeaderboardViewModel{
				SharedLoggedInViewModel: view.SharedLoggedInViewModel{
					SharedViewModel: view.NewSharedViewModel(h.config, &view.Messages{Error: criticalError}, r, user),
				},
			}
		}

		// regardless of page, always show own rank
		if user != nil && !leaderboard.HasUser(user.ID) {
			// but only if leaderboard spans multiple pages
			if count, err := h.leaderboardService.CountUsers(true); err == nil && count > int64(pageParams.PageSize) {
				if l, err := h.leaderboardService.GetByIntervalAndUser(h.leaderboardService.GetDefaultScope(), user.ID, true); err == nil && len(l) > 0 {
					leaderboard = append(leaderboard, l[0])
				}
			}
		}
	} else {
		if by, ok := allowedAggregations[byParam]; ok {
			leaderboard, err = h.leaderboardService.GetAggregatedByInterval(h.leaderboardService.GetDefaultScope(), &by, pageParams, true)
			if err != nil {
				conf.Log().Request(r).Error("error while fetching general leaderboard items", "error", err)
				return &view.LeaderboardViewModel{
					SharedLoggedInViewModel: view.SharedLoggedInViewModel{
						SharedViewModel: view.NewSharedViewModel(h.config, &view.Messages{Error: criticalError}, r, user),
					},
				}
			}

			// regardless of page, always show own rank
			if user != nil {
				// but only if leaderboard could, in theory, span multiple pages
				if count, err := h.leaderboardService.CountUsers(true); err == nil && count > int64(pageParams.PageSize) {
					if l, err := h.leaderboardService.GetAggregatedByIntervalAndUser(h.leaderboardService.GetDefaultScope(), user.ID, &by, true); err == nil {
						leaderboard.AddMany(l)
					} else {
						conf.Log().Request(r).Error("error while fetching own aggregated user leaderboard", "error", err)
					}
				}
			}

			userLeaderboards := slice.GroupWith[*models.LeaderboardItemRanked, string](leaderboard, func(item *models.LeaderboardItemRanked) string {
				return item.UserID
			})
			userLanguages = map[string][]string{}
			for u, items := range userLeaderboards {
				userLanguages[u] = models.Leaderboard(items).TopKeysByUser(models.SummaryLanguage, u)
			}

			topKeys = leaderboard.TopKeys(by)
			if len(topKeys) > 0 {
				if keyParam == "" {
					keyParam = topKeys[0]
				}
				keyParam = strings.ToLower(keyParam)
				leaderboard = leaderboard.TopByKey(by, keyParam)
			}
		} else {
			return &view.LeaderboardViewModel{
				SharedLoggedInViewModel: view.SharedLoggedInViewModel{
					SharedViewModel: view.NewSharedViewModel(h.config, &view.Messages{Error: fmt.Sprintf("unsupported aggregation '%s'", byParam)}, r, user),
				},
			}
		}
	}

	leaderboard.FilterEmpty()

	memberDashboardLinks := h.buildMemberDashboardLinks(user)

	vm := &view.LeaderboardViewModel{
		SharedLoggedInViewModel: view.SharedLoggedInViewModel{
			SharedViewModel: view.NewSharedViewModel(h.config, nil, r, user),
			User:            user,
		},
		Tab:                  tab,
		By:                   byParam,
		Key:                  keyParam,
		Items:                leaderboard,
		MemberDashboardLinks: memberDashboardLinks,
		UserLanguages:        userLanguages,
		TopKeys:              topKeys,
		IntervalLabel:        h.leaderboardService.GetDefaultScope().GetHumanReadable(),
		PageParams:           pageParams,
	}
	return routeutils.WithSessionMessages(vm, r, w)
}

// buildMemberDashboardLinks returns a map of userID -> dashboard URL
// for users that the viewer can access (as team owner or admin).
func (h *LeaderboardHandler) buildMemberDashboardLinks(viewer *models.User) map[string]string {
	links := make(map[string]string)
	if viewer == nil {
		return links
	}

	var teams []*models.Team
	if viewer.IsAdmin {
		teams, _ = h.teamService.GetAll()
	} else {
		teams, _ = h.teamService.GetByUser(viewer.ID)
	}

	for _, team := range teams {
		isOwner, _ := h.teamService.IsTeamOwner(team.ID, viewer.ID)
		if !viewer.IsAdmin && !isOwner {
			continue
		}
		members, err := h.teamService.GetMembers(team.ID)
		if err != nil {
			continue
		}
		for _, member := range members {
			if _, exists := links[member.UserID]; !exists {
				links[member.UserID] = fmt.Sprintf("teams/%s/members/%s", team.ID, member.UserID)
			}
		}
	}
	return links
}
