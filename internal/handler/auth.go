package handler

import (
	"log"
	"net/http"

	"github.com/c3d4r/app_scaffold/internal/auth"
)

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if h.cognito == nil {
		http.Error(w, "auth not configured", http.StatusInternalServerError)
		return
	}

	redirectURI := h.callbackURL
	if redirectURI == "" {
		redirectURI = schemeHost(r) + "/auth/callback"
	}

	if dest := r.URL.Query().Get("redirect"); dest != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "scaffold_redirect",
			Value:    dest,
			Path:     "/",
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   300,
		})
	}

	loginURL := h.cognito.LoginURL(redirectURI)
	http.Redirect(w, r, loginURL, http.StatusFound)
}

func (h *Handler) handleCallback(w http.ResponseWriter, r *http.Request) {
	if h.cognito == nil {
		http.Error(w, "auth not configured", http.StatusInternalServerError)
		return
	}

	if errParam := r.URL.Query().Get("error"); errParam != "" {
		desc := r.URL.Query().Get("error_description")
		log.Printf("auth callback: cognito error: %s: %s", errParam, desc)
		http.Error(w, "authentication failed", http.StatusUnauthorized)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	redirectURI := h.callbackURL
	if redirectURI == "" {
		redirectURI = schemeHost(r) + "/auth/callback"
	}

	tokens, err := h.cognito.ExchangeCode(r.Context(), code, redirectURI)
	if err != nil {
		log.Printf("auth callback: exchange code: %v", err)
		http.Error(w, "failed to exchange authorization code", http.StatusInternalServerError)
		return
	}

	claims, err := h.cognito.VerifyIDToken(r.Context(), tokens.IDToken)
	if err != nil {
		log.Printf("auth callback: verify id token: %v", err)
		http.Error(w, "failed to verify identity", http.StatusInternalServerError)
		return
	}

	username := claims.Username
	if username == "" {
		username = claims.Email
	}

	session := auth.NewSession(claims.Sub, username, claims.Email)
	if err := h.sessionStore.PutSession(r.Context(), session); err != nil {
		log.Printf("auth callback: save session: %v", err)
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "scaffold_session",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	dest := "/default"
	if cookie, err := r.Cookie("scaffold_redirect"); err == nil {
		dest = cookie.Value
		http.SetCookie(w, &http.Cookie{
			Name:     "scaffold_redirect",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
		})
	}

	http.Redirect(w, r, dest, http.StatusFound)
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("scaffold_session"); err == nil && h.sessionStore != nil {
		h.sessionStore.DeleteSession(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "scaffold_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/about", http.StatusFound)
}

func schemeHost(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
		scheme = fwd
	}
	return scheme + "://" + r.Host
}

func (h *Handler) handleSignInDev(w http.ResponseWriter, r *http.Request) {
	session := auth.NewSession("dev-user", "Developer", "dev@localhost")

	if err := h.sessionStore.PutSession(r.Context(), session); err != nil {
		log.Printf("dev sign-in: save session: %v", err)
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "scaffold_session",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	dest := r.URL.Query().Get("redirect")
	if dest == "" {
		dest = "/default"
	}
	http.Redirect(w, r, dest, http.StatusFound)
}

func (h *Handler) userFromSession(r *http.Request) *auth.Session {
	cookie, err := r.Cookie("scaffold_session")
	if err != nil {
		return nil
	}
	if h.sessionStore == nil {
		return nil
	}
	session, err := h.sessionStore.GetSession(r.Context(), cookie.Value)
	if err != nil || session == nil || session.IsExpired() {
		return nil
	}
	return session
}
