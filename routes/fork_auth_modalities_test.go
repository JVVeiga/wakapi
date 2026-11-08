package routes

import (
	"encoding/base64"
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

// Upstream 2.18 restricts api keys to api endpoints. These tests make sure the
// web routes added by this fork (teams, admin, language switch) follow suit.

var forkAuthAdmin = &models.User{ID: "forkadmin", ApiKey: "fork-admin-api-key", IsAdmin: true}

func newForkWebRouter(userServiceMock *mocks.UserServiceMock) chi.Router {
	router := chi.NewRouter()
	router.Use(middlewares.NewSharedDataMiddleware())
	NewTeamsHandler(userServiceMock, nil, nil, nil, nil, nil).RegisterRoutes(router)
	NewAdminHandler(userServiceMock, nil, nil, nil, nil, nil).RegisterRoutes(router)
	NewLanguageHandler(userServiceMock).RegisterRoutes(router)
	return router
}

func TestForkWebRoutes_RejectApiKeyAuth(t *testing.T) {
	config.Set(config.Empty())

	userServiceMock := new(mocks.UserServiceMock)
	userServiceMock.On("GetUserByKey", forkAuthAdmin.ApiKey, mock.Anything).Return(forkAuthAdmin, nil)
	userServiceMock.On("Update", mock.Anything).Return(forkAuthAdmin, nil)

	router := newForkWebRouter(userServiceMock)

	paths := []string{"/teams", "/teams/some-team", "/admin", "/admin/teams", "/admin/sites"}

	for _, path := range paths {
		t.Run(path+" via api_key query param", func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path+"?api_key="+forkAuthAdmin.ApiKey, nil)
			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusFound, rec.Code)
			assert.Equal(t, "/", rec.Header().Get("Location"))
		})

		t.Run(path+" via bearer header", func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Authorization", "Bearer "+base64.StdEncoding.EncodeToString([]byte(forkAuthAdmin.ApiKey)))
			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusFound, rec.Code)
			assert.Equal(t, "/", rec.Header().Get("Location"))
		})
	}

	t.Run("/lang does not resolve a principal from api key", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/lang?lang=en&api_key="+forkAuthAdmin.ApiKey, nil)
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusFound, rec.Code)
		userServiceMock.AssertNotCalled(t, "Update", mock.Anything)
	})

	userServiceMock.AssertNotCalled(t, "GetUserByKey", mock.Anything, mock.Anything)
}
