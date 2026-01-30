package middlewares

import (
	"net/http"

	conf "github.com/muety/wakapi/config"
)

type RequireAdminMiddleware struct {
	config *conf.Config
}

func NewRequireAdminMiddleware() *RequireAdminMiddleware {
	return &RequireAdminMiddleware{
		config: conf.Get(),
	}
}

func (m *RequireAdminMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetPrincipal(r)
		if user == nil || !user.IsAdmin {
			conf.Log().Request(r).Warn("unauthorized admin access attempt",
				"user", userIdOrAnonymous(user),
				"path", r.URL.Path,
			)
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(http.StatusText(http.StatusForbidden)))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func userIdOrAnonymous(user interface{ Identity() string }) string {
	if user == nil {
		return "anonymous"
	}
	return user.Identity()
}
