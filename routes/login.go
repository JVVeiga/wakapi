package routes

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dchest/captcha"
	"github.com/duke-git/lancet/v2/random"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/middlewares"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/models/view"
	routeutils "github.com/muety/wakapi/routes/utils"
	"github.com/muety/wakapi/services"
	"github.com/muety/wakapi/utils"
	i18n "github.com/muety/wakapi/views/i18n"
)

// TODO(oidc): tests (not only for oidc, but login in general)

type LoginHandler struct {
	config       *conf.Config
	userSrvc     services.IUserService
	mailSrvc     services.IMailService
	keyValueSrvc services.IKeyValueService
}

func NewLoginHandler(userService services.IUserService, mailService services.IMailService, keyValueService services.IKeyValueService) *LoginHandler {
	return &LoginHandler{
		config:       conf.Get(),
		userSrvc:     userService,
		mailSrvc:     mailService,
		keyValueSrvc: keyValueService,
	}
}

func (h *LoginHandler) RegisterRoutes(router chi.Router) {
	router.Get("/login", h.GetIndex)
	router.
		With(httprate.LimitByRealIP(h.config.Security.GetLoginMaxRate())).
		Post("/login", h.PostLogin)
	router.Get("/signup", h.GetSignup)
	router.
		With(httprate.LimitByRealIP(h.config.Security.GetSignupMaxRate())).
		Post("/signup", h.PostSignup)
	router.Get("/set-password", h.GetSetPassword)
	router.Post("/set-password", h.PostSetPassword)
	router.Get("/reset-password", h.GetResetPassword)
	router.
		With(httprate.LimitByRealIP(h.config.Security.GetPasswordResetMaxRate())).
		Post("/reset-password", h.PostResetPassword)
	router.Get("/oidc/{provider}/login", h.GetOidcLogin)
	router.Get("/oidc/{provider}/callback", h.GetOidcCallback)

	authMiddleware := middlewares.NewAuthenticateMiddleware(h.userSrvc).
		WithRedirectTarget(defaultErrorRedirectTarget()).
		WithRedirectErrorMessage("unauthorized").
		WithOptionalFor("/logout")

	logoutRouter := chi.NewRouter()
	logoutRouter.Use(authMiddleware.Handler)
	logoutRouter.Post("/", h.PostLogout)
	router.Mount("/logout", logoutRouter)
}

func (h *LoginHandler) GetIndex(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}

	if cookie, err := r.Cookie(models.AuthCookieKey); err == nil && cookie.Value != "" {
		http.Redirect(w, r, fmt.Sprintf("%s/summary", h.config.Server.BasePath), http.StatusFound)
		return
	}

	if !routeutils.HasErrorMessages(r) && h.config.Security.DisableLocalAuth && len(h.config.Security.OidcProviders) == 1 {
		http.Redirect(w, r, fmt.Sprintf("%s/oidc/%s/login", h.config.Server.BasePath, strings.ToLower(h.config.Security.OidcProviders[0].Name)), http.StatusFound)
		return
	}

	if h.config.Security.DisableLocalAuth && len(h.config.Security.OidcProviders) == 0 {
		lang := routeutils.ResolveLanguage(r, nil)
		routeutils.SetError(r, w, i18n.Translate(lang, "flash.no_auth_method"))
	}

	templates[conf.LoginTemplate].Execute(w, h.buildViewModel(r, w, false))
}

func (h *LoginHandler) PostLogin(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}

	if cookie, err := r.Cookie(models.AuthCookieKey); err == nil && cookie.Value != "" {
		http.Redirect(w, r, fmt.Sprintf("%s/summary", h.config.Server.BasePath), http.StatusFound)
		return
	}

	lang := routeutils.ResolveLanguage(r, nil)

	if h.config.Security.DisableLocalAuth {
		w.WriteHeader(http.StatusForbidden)
		templates[conf.LoginTemplate].Execute(w, h.buildViewModel(r, w, h.config.Security.SignupCaptcha).WithError(i18n.Translate(lang, "flash.local_auth_disabled")))
		return
	}

	var login models.Login
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates[conf.LoginTemplate].Execute(w, h.buildViewModel(r, w, false).WithError(i18n.Translate(lang, "flash.missing_parameters")))
		return
	}
	if err := loginDecoder.Decode(&login, r.PostForm); err != nil || login.Username == "" || login.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		templates[conf.LoginTemplate].Execute(w, h.buildViewModel(r, w, false).WithError(i18n.Translate(lang, "flash.missing_parameters")))
		return
	}

	user, err := h.userSrvc.GetUserById(login.Username)
	if err != nil {
		// try to get by email if given username is an email address (checked inside service)
		if user, err = h.userSrvc.GetUserByEmail(login.Username); err != nil {
			w.WriteHeader(http.StatusNotFound)
			templates[conf.LoginTemplate].Execute(w, h.buildViewModel(r, w, false).WithError(i18n.Translate(lang, "flash.resource_not_found")))
			return
		}
	}

	if !utils.ComparePassword(user.Password, login.Password, h.config.Security.PasswordSalt) {
		w.WriteHeader(http.StatusUnauthorized)
		templates[conf.LoginTemplate].Execute(w, h.buildViewModel(r, w, false).WithError(i18n.Translate(lang, "flash.invalid_credentials")))
		return
	}

	h.finishUserLogin(user, r, w)
	http.Redirect(w, r, fmt.Sprintf("%s/summary", h.config.Server.BasePath), http.StatusFound)
}

func (h *LoginHandler) PostLogout(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}

	if user := middlewares.GetPrincipal(r); user != nil {
		h.userSrvc.FlushUserCache(user.ID)
	}
	routeutils.ClearSession(r, w)                                    // clear all session data
	http.SetCookie(w, h.config.GetClearCookie(models.AuthCookieKey)) // clear auth token
	http.Redirect(w, r, fmt.Sprintf("%s/", h.config.Server.BasePath), http.StatusFound)
}

func (h *LoginHandler) GetSignup(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}

	if cookie, err := r.Cookie(models.AuthCookieKey); err == nil && cookie.Value != "" {
		http.Redirect(w, r, fmt.Sprintf("%s/summary", h.config.Server.BasePath), http.StatusFound)
		return
	}

	templates[conf.SignupTemplate].Execute(w, h.buildViewModel(r, w, h.config.Security.SignupCaptcha))
}

func (h *LoginHandler) PostSignup(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}

	lang := routeutils.ResolveLanguage(r, nil)

	var signup models.Signup
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates[conf.SignupTemplate].Execute(w, h.buildViewModel(r, w, h.config.Security.SignupCaptcha).WithError(i18n.Translate(lang, "flash.missing_parameters")))
		return
	}
	if err := signupDecoder.Decode(&signup, r.PostForm); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates[conf.SignupTemplate].Execute(w, h.buildViewModel(r, w, h.config.Security.SignupCaptcha).WithError(i18n.Translate(lang, "flash.missing_parameters")))
		return
	}

	if !h.config.IsDev() && !h.config.Security.AllowSignup && (!h.config.Security.InviteCodes || signup.InviteCode == "") {
		w.WriteHeader(http.StatusForbidden)
		templates[conf.SignupTemplate].Execute(w, h.buildViewModel(r, w, h.config.Security.SignupCaptcha).WithError(i18n.Translate(lang, "flash.signup_disabled")))
		return
	}

	if h.config.Security.DisableLocalAuth {
		w.WriteHeader(http.StatusForbidden)
		templates[conf.LoginTemplate].Execute(w, h.buildViewModel(r, w, h.config.Security.SignupCaptcha).WithError(i18n.Translate(lang, "flash.local_auth_disabled_signup")))
		return
	}

	if cookie, err := r.Cookie(models.AuthCookieKey); err == nil && cookie.Value != "" {
		http.Redirect(w, r, fmt.Sprintf("%s/summary", h.config.Server.BasePath), http.StatusFound)
		return
	}

	var invitedBy string
	var invitedDate time.Time
	var inviteCodeKey = fmt.Sprintf("%s_%s", conf.KeyInviteCode, signup.InviteCode)

	if signup.InviteCode != "" {
		if kv, _ := h.keyValueSrvc.GetString(inviteCodeKey); kv != nil && kv.Value != "" {
			if parts := strings.Split(kv.Value, ","); len(parts) == 2 {
				invitedBy = parts[0]
				invitedDate, _ = time.Parse(time.RFC3339, parts[1])
			}

			if err := h.keyValueSrvc.DeleteString(inviteCodeKey); err != nil {
				conf.Log().Error("failed to revoke invite code", "inviteCodeKey", inviteCodeKey, "error", err)
			}
		}
	}

	if signup.InviteCode != "" && time.Since(invitedDate) > 24*time.Hour {
		w.WriteHeader(http.StatusForbidden)
		templates[conf.SignupTemplate].Execute(w, h.buildViewModel(r, w, h.config.Security.SignupCaptcha).WithError(i18n.Translate(lang, "flash.invite_invalid")))
		return
	}

	signup.InvitedBy = invitedBy

	if !signup.IsValid() {
		w.WriteHeader(http.StatusBadRequest)
		errMsg := i18n.Translate(lang, "flash.invalid_parameters")
		if !models.ValidateUsername(signup.Username) {
			errMsg = i18n.Translate(lang, "flash.invalid_username_value")
		}
		templates[conf.SignupTemplate].Execute(w, h.buildViewModel(r, w, h.config.Security.SignupCaptcha).WithError(errMsg))
		return
	}

	numUsers, _ := h.userSrvc.Count()

	_, created, err := h.userSrvc.CreateOrGet(&signup, numUsers == 0)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		conf.Log().Request(r).Error("failed to create new user", "error", err)
		templates[conf.SignupTemplate].Execute(w, h.buildViewModel(r, w, h.config.Security.SignupCaptcha).WithError(i18n.Translate(lang, "flash.user_creation_failed")))
		return
	}
	if !created {
		w.WriteHeader(http.StatusConflict)
		templates[conf.SignupTemplate].Execute(w, h.buildViewModel(r, w, h.config.Security.SignupCaptcha).WithError(i18n.Translate(lang, "flash.user_exists")))
		return
	}

	routeutils.SetSuccess(r, w, i18n.Translate(lang, "flash.account_created"))
	http.Redirect(w, r, h.config.Server.BasePath, http.StatusFound)
}

func (h *LoginHandler) GetResetPassword(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}
	templates[conf.ResetPasswordTemplate].Execute(w, h.buildViewModel(r, w, false))
}

func (h *LoginHandler) GetSetPassword(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}

	lang := routeutils.ResolveLanguage(r, nil)

	values, _ := url.ParseQuery(r.URL.RawQuery)
	token := values.Get("token")
	if token == "" {
		w.WriteHeader(http.StatusUnauthorized)
		templates[conf.SetPasswordTemplate].Execute(w, h.buildViewModel(r, w, false).WithError(i18n.Translate(lang, "flash.invalid_token")))
		return
	}

	vm := &view.SetPasswordViewModel{
		LoginViewModel: *h.buildViewModel(r, w, false),
		Token:          token,
	}

	templates[conf.SetPasswordTemplate].Execute(w, vm)
}

func (h *LoginHandler) PostSetPassword(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}

	lang := routeutils.ResolveLanguage(r, nil)

	var setRequest models.SetPasswordRequest
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates[conf.SetPasswordTemplate].Execute(w, h.buildViewModel(r, w, false).WithError(i18n.Translate(lang, "flash.missing_parameters")))
		return
	}
	if err := signupDecoder.Decode(&setRequest, r.PostForm); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates[conf.SetPasswordTemplate].Execute(w, h.buildViewModel(r, w, false).WithError(i18n.Translate(lang, "flash.missing_parameters")))
		return
	}

	user, err := h.userSrvc.GetUserByResetToken(setRequest.Token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		templates[conf.SetPasswordTemplate].Execute(w, h.buildViewModel(r, w, false).WithError(i18n.Translate(lang, "flash.invalid_token")))
		return
	}

	if !setRequest.IsValid() {
		w.WriteHeader(http.StatusBadRequest)
		templates[conf.SetPasswordTemplate].Execute(w, h.buildViewModel(r, w, false).WithError(i18n.Translate(lang, "flash.invalid_parameters")))
		return
	}

	user.Password = setRequest.Password
	user.ResetToken = ""
	if hash, err := utils.HashPassword(user.Password, h.config.Security.PasswordSalt); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		conf.Log().Request(r).Error("failed to set new password", "error", err)
		templates[conf.SetPasswordTemplate].Execute(w, h.buildViewModel(r, w, false).WithError(i18n.Translate(lang, "flash.password_set_failed")))
		return
	} else {
		user.Password = hash
	}

	if _, err := h.userSrvc.Update(user); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		conf.Log().Request(r).Error("failed to save new password", "error", err)
		templates[conf.SetPasswordTemplate].Execute(w, h.buildViewModel(r, w, false).WithError(i18n.Translate(lang, "flash.password_save_failed")))
		return
	}

	routeutils.SetSuccess(r, w, i18n.Translate(lang, "flash.password_updated"))
	http.Redirect(w, r, fmt.Sprintf("%s/login", h.config.Server.BasePath), http.StatusFound)
}

func (h *LoginHandler) PostResetPassword(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}

	lang := routeutils.ResolveLanguage(r, nil)

	if !h.config.Mail.Enabled {
		w.WriteHeader(http.StatusNotImplemented)
		templates[conf.ResetPasswordTemplate].Execute(w, h.buildViewModel(r, w, false).WithError(i18n.Translate(lang, "flash.mailing_disabled")))
		return
	}

	var resetRequest models.ResetPasswordRequest
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates[conf.ResetPasswordTemplate].Execute(w, h.buildViewModel(r, w, false).WithError(i18n.Translate(lang, "flash.missing_parameters")))
		return
	}
	if err := resetPasswordDecoder.Decode(&resetRequest, r.PostForm); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates[conf.ResetPasswordTemplate].Execute(w, h.buildViewModel(r, w, false).WithError(i18n.Translate(lang, "flash.missing_parameters")))
		return
	}

	if user, err := h.userSrvc.GetUserByEmail(resetRequest.Email); user != nil && err == nil {
		if user.AuthType != "local" {
			conf.Log().Request(r).Warn("non-local user tried to reset password", "user", user.ID)
			w.WriteHeader(http.StatusInternalServerError)
			templates[conf.ResetPasswordTemplate].Execute(w, h.buildViewModel(r, w, false).WithError(i18n.Translate(lang, "flash.password_reset_failed")))
			return
		}

		if u, err := h.userSrvc.GenerateResetToken(user); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			conf.Log().Request(r).Error("failed to generate password reset token", "error", err)
			templates[conf.ResetPasswordTemplate].Execute(w, h.buildViewModel(r, w, false).WithError(i18n.Translate(lang, "flash.reset_token_failed")))
			return
		} else {
			go func(user *models.User, r *http.Request) {
				link := fmt.Sprintf("%s/set-password?token=%s", h.config.Server.GetPublicUrl(), user.ResetToken)
				if err := h.mailSrvc.SendPasswordReset(user, link); err != nil {
					conf.Log().Request(r).Error("failed to send password reset mail", "userID", user.ID, "error", err)
				} else {
					slog.Info("sent password reset mail", "userID", user.ID)
				}
			}(u, r)
		}
	} else {
		conf.Log().Request(r).Warn("password reset requested for unregistered address", "email", resetRequest.Email)
	}

	routeutils.SetSuccess(r, w, i18n.Translate(lang, "flash.email_sent"))
	http.Redirect(w, r, h.config.Server.BasePath, http.StatusFound)
}

func (h *LoginHandler) GetOidcLogin(w http.ResponseWriter, r *http.Request) {
	provider := h.getOidcProvider(w, r)
	if provider == nil {
		return // redirect done in previous method
	}
	state := routeutils.SetNewOidcState(r, w) // encoding state into session is fine, because session cookie is encrypted
	http.Redirect(w, r, provider.OAuth2.AuthCodeURL(state), http.StatusFound)
}

func (h *LoginHandler) GetOidcCallback(w http.ResponseWriter, r *http.Request) {
	provider := h.getOidcProvider(w, r)
	if provider == nil {
		return // redirect done in previous method
	}

	lang := routeutils.ResolveLanguage(r, nil)
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	// clear any existing id token on the session, just because
	routeutils.ClearOidcIdTokenPayload(r, w)

	// validate oauth state param
	savedState := routeutils.GetOidcState(r)
	if state == "" || savedState != state {
		errMsg := i18n.Translate(lang, "flash.oidc_invalid_state")
		conf.Log().Request(r).Error(errMsg, "saved_state", savedState, "state", state, "provider", provider.Name)
		routeutils.SetError(r, w, errMsg)
		http.Redirect(w, r, fmt.Sprintf("%s/login", h.config.Server.BasePath), http.StatusFound)
		return
	}
	routeutils.ClearOidcState(r, w)

	// exchange auth code for access token and id token
	authToken, err := provider.OAuth2.Exchange(r.Context(), code)
	if err != nil {
		errMsg := i18n.Translate(lang, "flash.oidc_exchange_failed")
		conf.Log().Request(r).Error(errMsg, "provider", provider.Name)
		routeutils.SetError(r, w, errMsg)
		http.Redirect(w, r, fmt.Sprintf("%s/login", h.config.Server.BasePath), http.StatusFound)
		return
	}

	// extract id token
	rawIdToken, ok := authToken.Extra("id_token").(string)
	if !ok {
		errMsg := i18n.Translate(lang, "flash.oidc_id_token_failed")
		conf.Log().Request(r).Error(errMsg, "provider", provider.Name)
		routeutils.SetError(r, w, errMsg)
		http.Redirect(w, r, fmt.Sprintf("%s/login", h.config.Server.BasePath), http.StatusFound)
		return
	}

	// verify id token
	idTokenPayload, err := routeutils.DecodeOidcIdToken(rawIdToken, provider, r.Context())
	if err != nil || idTokenPayload == nil {
		errMsg := i18n.Translate(lang, "flash.oidc_verify_failed")
		conf.Log().Request(r).Error(errMsg, "provider", provider.Name, "id_token", rawIdToken) // save to log, because does not grant any access
		routeutils.SetError(r, w, errMsg)
		http.Redirect(w, r, fmt.Sprintf("%s/login", h.config.Server.BasePath), http.StatusFound)
		return
	}

	user, err := h.userSrvc.GetUserByOidc(provider.Name, idTokenPayload.Subject)
	if err != nil {
		// create new user account
		if !h.config.IsDev() && !h.config.Security.OidcAllowSignup {
			routeutils.SetError(r, w, i18n.Translate(lang, "flash.signup_disabled"))
			http.Redirect(w, r, fmt.Sprintf("%s/login", h.config.Server.BasePath), http.StatusFound)
			return
		}

		signup := models.SignupFromOidcIdToken(idTokenPayload)
		if !signup.IsValid() {
			routeutils.SetError(r, w, i18n.Translate(lang, "flash.invalid_username"))
			http.Redirect(w, r, fmt.Sprintf("%s/login", h.config.Server.BasePath), http.StatusFound)
			return
		}

		if newUsername := h.coalesceExistingUser(signup.Username); newUsername != signup.Username {
			slog.Warn("username from id token already exist, using suffixed one instead", "username", newUsername)
			signup.Username = newUsername
		}

		slog.Info("creating new user from successful oidc authentication",
			"provider", signup.OidcProvider,
			"username", signup.Username,
			"email", signup.Email,
			"sub", signup.OidcSubject,
		)

		newUser, created, err := h.userSrvc.CreateOrGet(signup, false)
		if err != nil || !created {
			conf.Log().Request(r).Error("failed to create new user", "error", err, "provider", signup.OidcProvider, "username", signup.Username, "email", signup.Email)
			routeutils.SetError(r, w, i18n.Translate(lang, "flash.user_creation_failed"))
			http.Redirect(w, r, fmt.Sprintf("%s/login", h.config.Server.BasePath), http.StatusFound)
			return
		}
		user = newUser
	}

	routeutils.SetOidcIdTokenPayload(idTokenPayload, r, w) // save to session, only used by middleware for automatic redirection upon expiry
	h.finishUserLogin(user, r, w)
	http.Redirect(w, r, fmt.Sprintf("%s/summary", h.config.Server.BasePath), http.StatusFound)
}

func (h *LoginHandler) buildViewModel(r *http.Request, w http.ResponseWriter, withCaptcha bool) *view.LoginViewModel {
	numUsers, _ := h.userSrvc.Count()

	vm := &view.LoginViewModel{
		SharedViewModel:  view.NewSharedViewModel(h.config, nil, r, nil),
		TotalUsers:       int(numUsers),
		AllowSignup:      h.config.IsDev() || h.config.Security.AllowSignup,
		InviteCode:       r.URL.Query().Get("invite"),
		DisableLocalAuth: h.config.Security.DisableLocalAuth,
		OidcProviders: slice.Map(h.config.Security.ListOidcProviders(), func(i int, providerName string) view.LoginViewModelOidcProvider {
			provider, _ := conf.GetOidcProvider(providerName) // no error, because only using registered provider names
			return view.LoginViewModelOidcProvider{
				Name:        provider.Name,
				DisplayName: provider.DisplayName,
			}
		}),
	}

	if withCaptcha {
		vm.CaptchaId = captcha.New()
	}

	return routeutils.WithSessionMessages(vm, r, w)
}

func (h *LoginHandler) getOidcProvider(w http.ResponseWriter, r *http.Request) *conf.OidcProvider {
	providerName := chi.URLParam(r, "provider")
	provider, err := conf.GetOidcProvider(providerName)
	if err != nil {
		lang := routeutils.ResolveLanguage(r, nil)
		routeutils.SetError(r, w, fmt.Sprintf(i18n.Translate(lang, "flash.oidc_not_registered"), providerName))
		http.Redirect(w, r, fmt.Sprintf("%s/login", h.config.Server.BasePath), http.StatusFound)
		return nil
	}
	return provider
}

func (h *LoginHandler) finishUserLogin(user *models.User, r *http.Request, w http.ResponseWriter) {
	encoded, err := h.config.Security.SecureCookie.Encode(models.AuthCookieKey, user.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		conf.Log().Request(r).Error("failed to encode secure cookie", "error", err)
		lang := routeutils.ResolveLanguage(r, nil)
		templates[conf.LoginTemplate].Execute(w, h.buildViewModel(r, w, false).WithError(i18n.Translate(lang, "flash.internal_server_error")))
		return
	}

	user.LastLoggedInAt = models.CustomTime(time.Now())
	h.userSrvc.Update(user)

	http.SetCookie(w, h.config.CreateCookie(models.AuthCookieKey, encoded))
}

func (h *LoginHandler) coalesceExistingUser(username string) string {
	if u, _ := h.userSrvc.GetUserById(username); u != nil {
		return fmt.Sprintf("%s-%s", username, strings.ToLower(random.RandString(6)))
	}
	return username
}
