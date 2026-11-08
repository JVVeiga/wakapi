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
	"github.com/stretchr/testify/mock"
)

// Fork: admins and team owners/co-owners may read member data via the WakaTime API,
// including entity-filtered stats the member did not opt in to share publicly.
func TestStatsHandler_Get_FilterPermissions(t *testing.T) {
	config.Set(config.Empty())

	member := &models.User{ID: "member", ApiKey: "member-key", ShareDataMaxDays: -1, ShareProjects: false}
	admin := &models.User{ID: "admin", ApiKey: "admin-key", IsAdmin: true}
	owner := &models.User{ID: "owner", ApiKey: "owner-key"}
	stranger := &models.User{ID: "stranger", ApiKey: "stranger-key"}

	userServiceMock := new(mocks.UserServiceMock)
	for _, u := range []*models.User{member, admin, owner, stranger} {
		userServiceMock.On("GetUserById", u.ID).Return(u, nil)
		userServiceMock.On("GetUserByKey", u.ApiKey, false).Return(u, nil)
	}

	team := &models.Team{ID: "team1"}
	teamServiceMock := new(mocks.TeamServiceMock)
	teamServiceMock.On("GetByUser", member.ID).Return([]*models.Team{team}, nil)
	teamServiceMock.On("IsTeamOwnerOrCoOwner", team.ID, owner.ID).Return(true, nil)
	teamServiceMock.On("IsTeamOwnerOrCoOwner", team.ID, mock.Anything).Return(false, nil)

	summaryServiceMock := new(mocks.SummaryServiceMock)
	summaryServiceMock.On("Aliased", mock.Anything, mock.Anything, member, mock.Anything, mock.Anything, mock.Anything, false).
		Return(&models.Summary{UserID: member.ID}, nil)

	router := chi.NewRouter()
	apiRouter := chi.NewRouter()
	apiRouter.Use(middlewares.NewSharedDataMiddleware())
	router.Mount("/api", apiRouter)
	NewStatsHandler(userServiceMock, summaryServiceMock, teamServiceMock).RegisterRoutes(apiRouter)

	cases := []struct {
		name   string
		apiKey string
		status int
	}{
		{"admin can filter by project", admin.ApiKey, http.StatusOK},
		{"team owner can filter by project", owner.ApiKey, http.StatusOK},
		{"unrelated user cannot filter by unshared project", stranger.ApiKey, http.StatusForbidden},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/compat/wakatime/v1/users/member/stats/last_7_days?project=secret&api_key="+c.apiKey, nil)
			router.ServeHTTP(rec, req)
			assert.Equal(t, c.status, rec.Code, rec.Body.String())
		})
	}
}
