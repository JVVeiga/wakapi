package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/middlewares"
	routeutils "github.com/muety/wakapi/routes/utils"
	"github.com/muety/wakapi/services"
	i18n "github.com/muety/wakapi/views/i18n"
)

type LanguageHandler struct {
	config      *conf.Config
	userService services.IUserService
}

func NewLanguageHandler(userService services.IUserService) *LanguageHandler {
	return &LanguageHandler{
		config:      conf.Get(),
		userService: userService,
	}
}

func (h *LanguageHandler) RegisterRoutes(router chi.Router) {
	r := chi.NewRouter()
	r.Use(middlewares.NewAuthenticateMiddleware(h.userService).WithOptionalFor("/").Handler)
	r.Get("/", h.GetSwitch)
	router.Mount("/lang", r)
}

func (h *LanguageHandler) GetSwitch(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	if !i18n.IsSupported(lang) {
		lang = h.config.App.DefaultLanguage
	}

	// Set language cookie
	http.SetCookie(w, h.config.CreateCookie(routeutils.LangCookieName, lang))

	// Update user preference if logged in
	if user := middlewares.GetPrincipal(r); user != nil {
		user.Language = lang
		if _, err := h.userService.Update(user); err != nil {
			conf.Log().Request(r).Error("failed to update user language", "error", err)
		}
	}

	// Redirect back
	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = h.config.Server.BasePath + "/"
	}
	http.Redirect(w, r, referer, http.StatusFound)
}
