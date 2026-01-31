package utils

import (
	"net/http"

	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/models"
	i18n "github.com/muety/wakapi/views/i18n"
)

const LangCookieName = "wakapi_lang"

func ResolveLanguage(r *http.Request, user *models.User) string {
	if user != nil && user.Language != "" {
		return user.Language
	}
	if c, err := r.Cookie(LangCookieName); err == nil && i18n.IsSupported(c.Value) {
		return c.Value
	}
	return config.Get().App.DefaultLanguage
}
