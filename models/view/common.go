package view

import (
	"net/http"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/models"
	i18n "github.com/muety/wakapi/views/i18n"
)

type BasicViewModel interface {
	SetError(string)
	SetSuccess(string)
}

type Messages struct {
	Success string
	Error   string
}

type SharedViewModel struct {
	Messages
	LeaderboardEnabled bool
	InvitesEnabled     bool
	Language           string
}

type SharedLoggedInViewModel struct {
	SharedViewModel
	User *models.User
}

func NewSharedViewModel(c *conf.Config, messages *Messages, r *http.Request, user *models.User) SharedViewModel {
	lang := c.App.DefaultLanguage
	if user != nil && user.Language != "" {
		lang = user.Language
	} else if r != nil {
		if cookie, err := r.Cookie("wakapi_lang"); err == nil && i18n.IsSupported(cookie.Value) {
			lang = cookie.Value
		}
	}

	vm := SharedViewModel{
		LeaderboardEnabled: c.App.LeaderboardEnabled,
		InvitesEnabled:     c.Security.InviteCodes,
		Language:           lang,
	}
	if messages != nil {
		vm.Messages = *messages
	}
	return vm
}

func (m *Messages) SetError(message string) {
	m.Error = message
}

func (m *Messages) SetSuccess(message string) {
	m.Success = message
}

func (m SharedLoggedInViewModel) ApiKey() string {
	if m.User != nil {
		return m.User.ApiKey
	}
	return ""
}

// SetLanguage sets the language for translations
func (s *SharedViewModel) SetLanguage(lang string) {
	s.Language = lang
}

// T translates a key to the current language.
// Uses value receiver so it works in Go templates regardless of whether
// the view model is passed by value or pointer.
func (s SharedViewModel) T(key string) string {
	return i18n.Translate(s.Language, key)
}

// SupportedLanguages returns the list of available languages
func (s SharedViewModel) SupportedLanguages() []string {
	return i18n.SupportedLanguages()
}

// LanguageLabel returns the human-readable label for a language code
func (s SharedViewModel) LanguageLabel(lang string) string {
	return i18n.LanguageLabel(lang)
}
