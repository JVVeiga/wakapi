package routes

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/middlewares"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/models/view"
	routeutils "github.com/muety/wakapi/routes/utils"
	"github.com/muety/wakapi/services"
)

type AdminHandler struct {
	config           *conf.Config
	userSrvc         services.IUserService
	heartbeatSrvc    services.IHeartbeatService
	summarySrvc      services.ISummaryService
	apiKeySrvc       services.IApiKeyService
	teamSrvc         services.ITeamService
}

func NewAdminHandler(
	userService services.IUserService,
	heartbeatService services.IHeartbeatService,
	summaryService services.ISummaryService,
	apiKeyService services.IApiKeyService,
	teamService services.ITeamService,
) *AdminHandler {
	return &AdminHandler{
		config:        conf.Get(),
		userSrvc:      userService,
		heartbeatSrvc: heartbeatService,
		summarySrvc:   summaryService,
		apiKeySrvc:    apiKeyService,
		teamSrvc:      teamService,
	}
}

func (h *AdminHandler) RegisterRoutes(router chi.Router) {
	r := chi.NewRouter()

	authMiddleware := middlewares.NewAuthenticateMiddleware(h.userSrvc)
	authMiddleware = authMiddleware.WithRedirectTarget(defaultErrorRedirectTarget())
	authMiddleware = authMiddleware.WithRedirectErrorMessage("unauthorized")
	adminMiddleware := middlewares.NewRequireAdminMiddleware()

	r.Use(authMiddleware.Handler)
	r.Use(adminMiddleware.Handler)
	r.Get("/", h.GetDashboard)
	r.Get("/users/{id}", h.GetUserDetail)
	r.Post("/users/{id}", h.PostUserAction)
	r.Get("/teams", h.GetTeams)
	r.Post("/teams", h.PostCreateTeam)
	r.Get("/teams/{id}", h.GetTeamDetail)
	r.Post("/teams/{id}", h.PostTeamAction)
	r.Post("/teams/{id}/members", h.PostTeamMemberAction)

	router.Mount("/admin", r)
}

const adminUsersPageSize = 15

func (h *AdminHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}

	user := middlewares.GetPrincipal(r)

	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}

	totalUsers, _ := h.userSrvc.Count()
	onlineUsers, _ := h.userSrvc.CountCurrentlyOnline()
	totalHeartbeats, _ := h.heartbeatSrvc.Count(false)

	activeUsers := 0
	if active, err := h.userSrvc.GetActive(false); err == nil {
		activeUsers = len(active)
	}

	totalPages := int((totalUsers + int64(adminUsersPageSize) - 1) / int64(adminUsersPageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	users, _ := h.userSrvc.GetAllPaginated(page, adminUsersPageSize)

	vm := &view.AdminDashboardViewModel{
		SharedLoggedInViewModel: view.SharedLoggedInViewModel{
			SharedViewModel: view.NewSharedViewModel(h.config, nil),
			User:            user,
		},
		TotalUsers:      totalUsers,
		ActiveUsers:     activeUsers,
		OnlineUsers:     onlineUsers,
		TotalHeartbeats: totalHeartbeats,
		Users:           users,
		Page:            page,
		PageSize:        adminUsersPageSize,
		TotalPages:      totalPages,
	}

	if err := templates[conf.AdminDashboardTemplate].Execute(w, routeutils.WithSessionMessages(vm, r, w)); err != nil {
		conf.Log().Request(r).Error("failed to render admin dashboard", "error", err)
	}
}

func (h *AdminHandler) GetUserDetail(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}

	adminUser := middlewares.GetPrincipal(r)
	userId := chi.URLParam(r, "id")

	targetUser, err := h.userSrvc.GetUserById(userId)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("%s/admin", h.config.Server.BasePath), http.StatusFound)
		return
	}

	apiKeys, _ := h.apiKeySrvc.GetByUser(targetUser.ID)
	userTeams, _ := h.teamSrvc.GetByUser(targetUser.ID)

	totalTime := ""
	lastActivity := ""
	if lastHb, err := h.heartbeatSrvc.GetLastByUser(targetUser); err == nil && !lastHb.IsZero() {
		lastActivity = helpers.FormatDateTimeHuman(lastHb)
	}
	if count, err := h.heartbeatSrvc.CountByUser(targetUser); err == nil {
		totalTime = fmt.Sprintf("%d heartbeats", count)
	}

	vm := &view.AdminUserDetailViewModel{
		SharedLoggedInViewModel: view.SharedLoggedInViewModel{
			SharedViewModel: view.NewSharedViewModel(h.config, nil),
			User:            adminUser,
		},
		TargetUser:    targetUser,
		ApiKeys:       apiKeys,
		MaskedMainKey: targetUser.ApiKey,
		TotalTime:     totalTime,
		LastActivity:  lastActivity,
		UserTeams:     userTeams,
	}

	if err := templates[conf.AdminUserDetailTemplate].Execute(w, routeutils.WithSessionMessages(vm, r, w)); err != nil {
		conf.Log().Request(r).Error("failed to render admin user detail", "error", err)
	}
}

func (h *AdminHandler) PostUserAction(w http.ResponseWriter, r *http.Request) {
	adminUser := middlewares.GetPrincipal(r)
	userId := chi.URLParam(r, "id")
	action := r.FormValue("action")

	targetUser, err := h.userSrvc.GetUserById(userId)
	if err != nil {
		routeutils.SetError(r, w, "user not found")
		http.Redirect(w, r, fmt.Sprintf("%s/admin", h.config.Server.BasePath), http.StatusFound)
		return
	}

	switch strings.ToLower(action) {
	case "promote_admin":
		if _, err := h.userSrvc.SetAdmin(targetUser, true); err != nil {
			routeutils.SetError(r, w, "failed to promote user")
		} else {
			conf.Log().Info("user promoted to admin",
				"admin", adminUser.ID,
				"target", targetUser.ID,
			)
			routeutils.SetSuccess(r, w, fmt.Sprintf("User '%s' promoted to admin", targetUser.ID))
		}
	case "demote_admin":
		// prevent self-demotion
		if adminUser.ID == targetUser.ID {
			routeutils.SetError(r, w, "cannot demote yourself")
		} else {
			// prevent removing the last admin
			allUsers, _ := h.userSrvc.GetAll()
			adminCount := 0
			for _, u := range allUsers {
				if u.IsAdmin {
					adminCount++
				}
			}
			if adminCount <= 1 {
				routeutils.SetError(r, w, "cannot remove the last admin")
			} else if _, err := h.userSrvc.SetAdmin(targetUser, false); err != nil {
				routeutils.SetError(r, w, "failed to demote user")
			} else {
				conf.Log().Info("user demoted from admin",
					"admin", adminUser.ID,
					"target", targetUser.ID,
				)
				routeutils.SetSuccess(r, w, fmt.Sprintf("User '%s' demoted from admin", targetUser.ID))
			}
		}
	default:
		routeutils.SetError(r, w, "unknown action")
	}

	http.Redirect(w, r, fmt.Sprintf("%s/admin/users/%s", h.config.Server.BasePath, userId), http.StatusFound)
}

func (h *AdminHandler) GetTeams(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}

	adminUser := middlewares.GetPrincipal(r)

	teams, err := h.teamSrvc.GetAll()
	if err != nil {
		conf.Log().Request(r).Error("failed to get teams", "error", err)
		teams = []*models.Team{}
	}

	teamMemberCounts := make(map[string]int64)
	for _, team := range teams {
		if count, err := h.teamSrvc.CountMembers(team.ID); err == nil {
			teamMemberCounts[team.ID] = count
		}
	}

	allUsers, _ := h.userSrvc.GetAll()

	vm := &view.AdminTeamsViewModel{
		SharedLoggedInViewModel: view.SharedLoggedInViewModel{
			SharedViewModel: view.NewSharedViewModel(h.config, nil),
			User:            adminUser,
		},
		Teams:            teams,
		TeamMemberCounts: teamMemberCounts,
		Users:            allUsers,
	}

	if err := templates[conf.AdminTeamsTemplate].Execute(w, routeutils.WithSessionMessages(vm, r, w)); err != nil {
		conf.Log().Request(r).Error("failed to render admin teams", "error", err)
	}
}

func (h *AdminHandler) PostCreateTeam(w http.ResponseWriter, r *http.Request) {
	adminUser := middlewares.GetPrincipal(r)
	name := r.FormValue("name")
	description := r.FormValue("description")
	ownerID := r.FormValue("owner_id")

	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	if name == "" || ownerID == "" {
		routeutils.SetError(r, w, "name and owner are required")
		http.Redirect(w, r, fmt.Sprintf("%s/admin/teams", h.config.Server.BasePath), http.StatusFound)
		return
	}

	if _, err := h.userSrvc.GetUserById(ownerID); err != nil {
		routeutils.SetError(r, w, "owner user not found")
		http.Redirect(w, r, fmt.Sprintf("%s/admin/teams", h.config.Server.BasePath), http.StatusFound)
		return
	}

	team, err := h.teamSrvc.CreateTeam(name, description, ownerID)
	if err != nil {
		routeutils.SetError(r, w, "failed to create team")
		http.Redirect(w, r, fmt.Sprintf("%s/admin/teams", h.config.Server.BasePath), http.StatusFound)
		return
	}

	conf.Log().Info("team created",
		"admin", adminUser.ID,
		"team_id", team.ID,
		"team_name", team.Name,
	)

	routeutils.SetSuccess(r, w, fmt.Sprintf("Team '%s' created successfully", team.Name))
	http.Redirect(w, r, fmt.Sprintf("%s/admin/teams/%s", h.config.Server.BasePath, team.ID), http.StatusFound)
}

func (h *AdminHandler) GetTeamDetail(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}

	adminUser := middlewares.GetPrincipal(r)
	teamID := chi.URLParam(r, "id")

	team, err := h.teamSrvc.GetByID(teamID)
	if err != nil {
		routeutils.SetError(r, w, "team not found")
		http.Redirect(w, r, fmt.Sprintf("%s/admin/teams", h.config.Server.BasePath), http.StatusFound)
		return
	}

	members, _ := h.teamSrvc.GetMembers(teamID)
	allUsers, _ := h.userSrvc.GetAll()

	vm := &view.AdminTeamDetailViewModel{
		SharedLoggedInViewModel: view.SharedLoggedInViewModel{
			SharedViewModel: view.NewSharedViewModel(h.config, nil),
			User:            adminUser,
		},
		Team:     team,
		Members:  members,
		AllUsers: allUsers,
	}

	if err := templates[conf.AdminTeamDetailTemplate].Execute(w, routeutils.WithSessionMessages(vm, r, w)); err != nil {
		conf.Log().Request(r).Error("failed to render admin team detail", "error", err)
	}
}

func (h *AdminHandler) PostTeamAction(w http.ResponseWriter, r *http.Request) {
	adminUser := middlewares.GetPrincipal(r)
	teamID := chi.URLParam(r, "id")
	action := r.FormValue("action")

	team, err := h.teamSrvc.GetByID(teamID)
	if err != nil {
		routeutils.SetError(r, w, "team not found")
		http.Redirect(w, r, fmt.Sprintf("%s/admin/teams", h.config.Server.BasePath), http.StatusFound)
		return
	}

	switch strings.ToLower(action) {
	case "delete":
		if err := h.teamSrvc.DeleteTeam(teamID); err != nil {
			routeutils.SetError(r, w, "failed to delete team")
		} else {
			conf.Log().Info("team deleted",
				"admin", adminUser.ID,
				"team_id", teamID,
			)
			routeutils.SetSuccess(r, w, fmt.Sprintf("Team '%s' deleted", team.Name))
		}
		http.Redirect(w, r, fmt.Sprintf("%s/admin/teams", h.config.Server.BasePath), http.StatusFound)
		return

	case "update":
		newName := strings.TrimSpace(r.FormValue("name"))
		newDescription := strings.TrimSpace(r.FormValue("description"))
		if newName == "" {
			routeutils.SetError(r, w, "team name is required")
			break
		}
		team.Name = newName
		team.Description = newDescription
		if _, err := h.teamSrvc.UpdateTeam(team); err != nil {
			routeutils.SetError(r, w, "failed to update team")
		} else {
			conf.Log().Info("team updated",
				"admin", adminUser.ID,
				"team_id", teamID,
			)
			routeutils.SetSuccess(r, w, "Team updated successfully")
		}

	default:
		routeutils.SetError(r, w, "unknown action")
	}

	http.Redirect(w, r, fmt.Sprintf("%s/admin/teams/%s", h.config.Server.BasePath, teamID), http.StatusFound)
}

func (h *AdminHandler) PostTeamMemberAction(w http.ResponseWriter, r *http.Request) {
	adminUser := middlewares.GetPrincipal(r)
	teamID := chi.URLParam(r, "id")
	action := r.FormValue("action")
	userID := r.FormValue("user_id")

	if userID == "" {
		routeutils.SetError(r, w, "user is required")
		http.Redirect(w, r, fmt.Sprintf("%s/admin/teams/%s", h.config.Server.BasePath, teamID), http.StatusFound)
		return
	}

	if _, err := h.teamSrvc.GetByID(teamID); err != nil {
		routeutils.SetError(r, w, "team not found")
		http.Redirect(w, r, fmt.Sprintf("%s/admin/teams", h.config.Server.BasePath), http.StatusFound)
		return
	}

	switch strings.ToLower(action) {
	case "add":
		if _, err := h.userSrvc.GetUserById(userID); err != nil {
			routeutils.SetError(r, w, "user not found")
			http.Redirect(w, r, fmt.Sprintf("%s/admin/teams/%s", h.config.Server.BasePath, teamID), http.StatusFound)
			return
		}
		role := r.FormValue("role")
		if role == "" {
			role = models.TeamRoleMember
		}
		if _, err := h.teamSrvc.AddMember(teamID, userID, role); err != nil {
			conf.Log().Request(r).Error("failed to add team member", "error", err)
			routeutils.SetError(r, w, "failed to add member")
		} else {
			conf.Log().Info("team member added",
				"admin", adminUser.ID,
				"team_id", teamID,
				"user_id", userID,
			)
			routeutils.SetSuccess(r, w, "Member added successfully")
		}

	case "remove":
		if err := h.teamSrvc.RemoveMember(teamID, userID); err != nil {
			conf.Log().Request(r).Error("failed to remove team member", "error", err)
			routeutils.SetError(r, w, "failed to remove member")
		} else {
			conf.Log().Info("team member removed",
				"admin", adminUser.ID,
				"team_id", teamID,
				"user_id", userID,
			)
			routeutils.SetSuccess(r, w, "Member removed successfully")
		}

	default:
		routeutils.SetError(r, w, "unknown action")
	}

	http.Redirect(w, r, fmt.Sprintf("%s/admin/teams/%s", h.config.Server.BasePath, teamID), http.StatusFound)
}
