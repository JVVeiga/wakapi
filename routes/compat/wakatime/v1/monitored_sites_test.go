package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/middlewares"
	"github.com/muety/wakapi/mocks"
	"github.com/muety/wakapi/models"
	"github.com/stretchr/testify/assert"
)

type monitoredSiteServiceStub struct{}

func (monitoredSiteServiceStub) GetAll() ([]*models.MonitoredSite, error) {
	return []*models.MonitoredSite{{URL: "https://example.com", Label: "Example"}}, nil
}
func (monitoredSiteServiceStub) GetByID(uint) (*models.MonitoredSite, error) { return nil, nil }
func (monitoredSiteServiceStub) Create(s *models.MonitoredSite) (*models.MonitoredSite, error) {
	return s, nil
}
func (monitoredSiteServiceStub) Update(s *models.MonitoredSite) (*models.MonitoredSite, error) {
	return s, nil
}
func (monitoredSiteServiceStub) Delete(uint) error   { return nil }
func (monitoredSiteServiceStub) Count() (int, error) { return 1, nil }

// Monitored sites is an api endpoint consumed with the user's api key; upstream 2.18 restricts
// api keys to api endpoints, so it must keep accepting them.
func TestMonitoredSitesHandler_Get_AcceptsApiKey(t *testing.T) {
	config.Set(config.Empty())

	user := &models.User{ID: "someone", ApiKey: "someone-key"}
	userServiceMock := new(mocks.UserServiceMock)
	userServiceMock.On("GetUserByKey", user.ApiKey, false).Return(user, nil)

	router := chi.NewRouter()
	apiRouter := chi.NewRouter()
	apiRouter.Use(middlewares.NewSharedDataMiddleware())
	router.Mount("/api", apiRouter)
	NewMonitoredSitesHandler(userServiceMock, monitoredSiteServiceStub{}).RegisterRoutes(apiRouter)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/compat/wakatime/v1/monitored-sites?api_key="+user.ApiKey, nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "https://example.com")
}
