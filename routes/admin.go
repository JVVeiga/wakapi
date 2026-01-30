package routes

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/middlewares"
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
}

func NewAdminHandler(
	userService services.IUserService,
	heartbeatService services.IHeartbeatService,
	summaryService services.ISummaryService,
	apiKeyService services.IApiKeyService,
) *AdminHandler {
	return &AdminHandler{
		config:        conf.Get(),
		userSrvc:      userService,
		heartbeatSrvc: heartbeatService,
		summarySrvc:   summaryService,
		apiKeySrvc:    apiKeyService,
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

	router.Mount("/admin", r)
}

func (h *AdminHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}

	user := middlewares.GetPrincipal(r)

	totalUsers, _ := h.userSrvc.Count()
	onlineUsers, _ := h.userSrvc.CountCurrentlyOnline()
	totalHeartbeats, _ := h.heartbeatSrvc.Count(false)

	activeUsers := 0
	if active, err := h.userSrvc.GetActive(false); err == nil {
		activeUsers = len(active)
	}

	allUsers, _ := h.userSrvc.GetAll()

	vm := &view.AdminDashboardViewModel{
		SharedLoggedInViewModel: view.SharedLoggedInViewModel{
			SharedViewModel: view.NewSharedViewModel(h.config, nil),
			User:            user,
		},
		TotalUsers:      totalUsers,
		ActiveUsers:     activeUsers,
		OnlineUsers:     onlineUsers,
		TotalHeartbeats: totalHeartbeats,
		Users:           allUsers,
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
