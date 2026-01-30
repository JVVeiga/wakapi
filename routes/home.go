package routes

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/schema"
	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/middlewares"
	"github.com/muety/wakapi/services"
	"net/http"
)

type HomeHandler struct {
	config       *conf.Config
	userSrvc     services.IUserService
	keyValueSrvc services.IKeyValueService
}

var loginDecoder = schema.NewDecoder()
var signupDecoder = schema.NewDecoder()
var resetPasswordDecoder = schema.NewDecoder()

func NewHomeHandler(userService services.IUserService, keyValueService services.IKeyValueService) *HomeHandler {
	return &HomeHandler{
		config:       conf.Get(),
		userSrvc:     userService,
		keyValueSrvc: keyValueService,
	}
}

func (h *HomeHandler) RegisterRoutes(router chi.Router) {
	router.Group(func(r chi.Router) {
		r.Use(middlewares.NewAuthenticateMiddleware(h.userSrvc).WithOptionalFor("/").Handler)
		r.Get("/", h.GetIndex)
	})
}

func (h *HomeHandler) GetIndex(w http.ResponseWriter, r *http.Request) {
	if user := middlewares.GetPrincipal(r); user != nil {
		http.Redirect(w, r, fmt.Sprintf("%s/summary", h.config.Server.BasePath), http.StatusFound)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("%s/login", h.config.Server.BasePath), http.StatusFound)
}

