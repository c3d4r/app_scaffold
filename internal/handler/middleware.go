package handler

import (
	"net/http"
	"net/url"
)

func (h *Handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.sessionStore == nil {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("scaffold_session")
		if err != nil {
			if h.cognito == nil {
				redirectToDevSignIn(w, r)
			} else {
				redirectToLogin(w, r)
			}
			return
		}

		session, err := h.sessionStore.GetSession(r.Context(), cookie.Value)
		if err != nil || session == nil || session.IsExpired() {
			clearSessionCookie(w)
			if h.cognito == nil {
				redirectToDevSignIn(w, r)
			} else {
				redirectToLogin(w, r)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	dest := "/auth/login?redirect=" + url.QueryEscape(r.URL.String())
	http.Redirect(w, r, dest, http.StatusFound)
}

func redirectToDevSignIn(w http.ResponseWriter, r *http.Request) {
	dest := "/auth/dev-sign-in?redirect=" + url.QueryEscape(r.URL.String())
	http.Redirect(w, r, dest, http.StatusFound)
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "scaffold_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}
