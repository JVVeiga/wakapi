package routes

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/middlewares"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/models/view"
	routeutils "github.com/muety/wakapi/routes/utils"
	"github.com/muety/wakapi/services"
)

type TeamsHandler struct {
	config      *conf.Config
	userSrvc    services.IUserService
	teamSrvc    services.ITeamService
	summarySrvc services.ISummaryService
}

func NewTeamsHandler(
	userService services.IUserService,
	teamService services.ITeamService,
	summaryService services.ISummaryService,
) *TeamsHandler {
	return &TeamsHandler{
		config:      conf.Get(),
		userSrvc:    userService,
		teamSrvc:    teamService,
		summarySrvc: summaryService,
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
			SharedViewModel: view.NewSharedViewModel(h.config, nil),
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

	team, err := h.teamSrvc.GetByID(teamID)
	if err != nil {
		routeutils.SetError(r, w, "team not found")
		http.Redirect(w, r, fmt.Sprintf("%s/teams", h.config.Server.BasePath), http.StatusFound)
		return
	}

	if !user.IsAdmin {
		isMember, err := h.teamSrvc.IsTeamMember(teamID, user.ID)
		if err != nil {
			conf.Log().Request(r).Error("failed to check team membership", "error", err, "team", teamID, "user", user.ID)
			routeutils.SetError(r, w, "internal error")
			http.Redirect(w, r, fmt.Sprintf("%s/teams", h.config.Server.BasePath), http.StatusFound)
			return
		}
		if !isMember {
			routeutils.SetError(r, w, "you are not a member of this team")
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
			SharedViewModel: view.NewSharedViewModel(h.config, nil),
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
