package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/middlewares"
	"github.com/muety/wakapi/services"
)

type MonitoredSitesHandler struct {
	config            *conf.Config
	userSrvc          services.IUserService
	monitoredSiteSrvc services.IMonitoredSiteService
}

func NewMonitoredSitesHandler(
	userService services.IUserService,
	monitoredSiteService services.IMonitoredSiteService,
) *MonitoredSitesHandler {
	return &MonitoredSitesHandler{
		config:            conf.Get(),
		userSrvc:          userService,
		monitoredSiteSrvc: monitoredSiteService,
	}
}

func (h *MonitoredSitesHandler) RegisterRoutes(router chi.Router) {
	router.Group(func(r chi.Router) {
		r.Use(middlewares.NewAuthenticateMiddleware(h.userSrvc).Handler)
		r.Get("/compat/wakatime/v1/monitored-sites", h.Get)
	})
}

type MonitoredSitesResponse struct {
	Total int                     `json:"total"`
	Sites []MonitoredSiteResponse `json:"sites"`
}

type MonitoredSiteResponse struct {
	URL   string `json:"url"`
	Label string `json:"label"`
}

// @Summary Retrieve monitored sites
// @Description Returns a list of all monitored sites with their URLs and labels
// @ID get-monitored-sites
// @Tags wakatime
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MonitoredSitesResponse
// @Router /compat/wakatime/v1/monitored-sites [get]
func (h *MonitoredSitesHandler) Get(w http.ResponseWriter, r *http.Request) {
	sites, err := h.monitoredSiteSrvc.GetAll()
	if err != nil {
		conf.Log().Request(r).Error("failed to get monitored sites", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	siteResponses := make([]MonitoredSiteResponse, len(sites))
	for i, site := range sites {
		siteResponses[i] = MonitoredSiteResponse{
			URL:   site.URL,
			Label: site.Label,
		}
	}

	response := MonitoredSitesResponse{
		Total: len(sites),
		Sites: siteResponses,
	}

	helpers.RespondJSON(w, r, http.StatusOK, response)
}
