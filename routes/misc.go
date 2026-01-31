package routes

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	conf "github.com/muety/wakapi/config"
	routeutils "github.com/muety/wakapi/routes/utils"
	"github.com/muety/wakapi/services"
	i18n "github.com/muety/wakapi/views/i18n"
)

// TODO(oidc): tests (not only for oidc, but login in general)

type MiscHandler struct {
	config   *conf.Config
	userSrvc services.IUserService
}

func NewMiscHandler(userService services.IUserService) *MiscHandler {
	return &MiscHandler{
		config:   conf.Get(),
		userSrvc: userService,
	}
}

func (h *MiscHandler) RegisterRoutes(router chi.Router) {
	router.Get("/unsubscribe", h.GetUnsubscribe)
}

func (h *MiscHandler) GetUnsubscribe(w http.ResponseWriter, r *http.Request) {
	lang := routeutils.ResolveLanguage(r, nil)

	token := r.URL.Query().Get("token")
	if token == "" {
		routeutils.SetError(r, w, i18n.Translate(lang, "flash.missing_token"))
		http.Redirect(w, r, fmt.Sprintf("%s", h.config.Server.BasePath), http.StatusFound)
		return
	}

	user, err := h.userSrvc.GetUserByUnsubscribeToken(token)
	if err != nil {
		routeutils.SetError(r, w, i18n.Translate(lang, "flash.invalid_token_param"))
		http.Redirect(w, r, fmt.Sprintf("%s", h.config.Server.BasePath), http.StatusFound)
		return
	}

	lang = routeutils.ResolveLanguage(r, user)

	user.ReportsWeekly = false
	if _, err := h.userSrvc.Update(user); err != nil {
		conf.Log().Request(r).Error("failed to unsubscribe user from weekly reports", "user", user.ID, "error", err)
		routeutils.SetError(r, w, i18n.Translate(lang, "flash.update_preferences_failed"))
		http.Redirect(w, r, fmt.Sprintf("%s", h.config.Server.BasePath), http.StatusFound)
		return
	}

	routeutils.SetSuccess(r, w, i18n.Translate(lang, "flash.unsubscribed_reports"))
	http.Redirect(w, r, fmt.Sprintf("%s", h.config.Server.BasePath), http.StatusFound)
}
