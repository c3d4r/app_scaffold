package handler

import (
	"log"
	"net/http"
	"strings"

	"github.com/c3d4r/app_scaffold/internal/auth"
	"github.com/c3d4r/app_scaffold/internal/template"
)

func (h *Handler) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if h.cognitoClient == nil {
			http.Redirect(w, r, "/auth/dev-sign-in", http.StatusFound)
			return
		}
		template.LoginPage("").Render(r.Context(), w)
		return
	}

	if h.cognitoClient == nil {
		http.Error(w, "auth not configured", http.StatusInternalServerError)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if username == "" || password == "" {
		template.LoginPage("Username and password are required.").Render(r.Context(), w)
		return
	}

	result, challenge, err := h.cognitoClient.SignIn(r.Context(), username, password)
	if err != nil {
		log.Printf("sign in error for %q: %v", username, err)
		template.LoginPage("Invalid username or password.").Render(r.Context(), w)
		return
	}

	if challenge != nil {
		template.LoginPage("Account requires a password change. Please use a different sign-in method.").Render(r.Context(), w)
		return
	}

	claims, err := h.cognito.VerifyIDToken(r.Context(), result.IDToken)
	if err != nil {
		log.Printf("verify id token: %v", err)
		template.LoginPage("Failed to verify identity.").Render(r.Context(), w)
		return
	}

	displayName := claims.Username
	if displayName == "" {
		displayName = claims.Email
	}

	session := auth.NewSession(claims.Sub, displayName, claims.Email)
	if err := h.sessionStore.PutSession(r.Context(), session); err != nil {
		log.Printf("save session: %v", err)
		template.LoginPage("Failed to create session.").Render(r.Context(), w)
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

	dest := r.URL.Query().Get("redirect")
	if dest == "" {
		dest = "/default"
	}
	http.Redirect(w, r, dest, http.StatusFound)
}

func (h *Handler) handleSignupPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if h.cognitoClient == nil {
			http.Error(w, "auth not configured", http.StatusInternalServerError)
			return
		}
		template.SignupPage("").Render(r.Context(), w)
		return
	}

	if h.cognitoClient == nil {
		http.Error(w, "auth not configured", http.StatusInternalServerError)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	if username == "" || email == "" || password == "" {
		template.SignupPage("All fields are required.").Render(r.Context(), w)
		return
	}

	if len(password) < 8 {
		template.SignupPage("Password must be at least 8 characters.").Render(r.Context(), w)
		return
	}

	if err := h.cognitoClient.SignUp(r.Context(), username, password, email); err != nil {
		log.Printf("sign up error for %q: %v", username, err)
		msg := "Sign up failed. Username may already exist."
		if strings.Contains(err.Error(), "UsernameExistsException") {
			msg = "An account with that username already exists."
		}
		template.SignupPage(msg).Render(r.Context(), w)
		return
	}

	dest := "/auth/confirm?username=" + username
	log.Printf("user %q signed up, redirecting to confirm", username)
	http.Redirect(w, r, dest, http.StatusFound)
}

func (h *Handler) handleConfirmPage(w http.ResponseWriter, r *http.Request) {
	if h.cognitoClient == nil {
		http.Error(w, "auth not configured", http.StatusInternalServerError)
		return
	}

	username := r.URL.Query().Get("username")
	if r.Method == http.MethodGet {
		if username == "" {
			http.Error(w, "missing username", http.StatusBadRequest)
			return
		}
		template.ConfirmPage(username, "", "").Render(r.Context(), w)
		return
	}

	code := strings.TrimSpace(r.FormValue("code"))

	if username == "" || code == "" {
		template.ConfirmPage(username, "Username and code are required.", "").Render(r.Context(), w)
		return
	}

	if err := h.cognitoClient.ConfirmSignUp(r.Context(), username, code); err != nil {
		log.Printf("confirm error for %q: %v", username, err)
		msg := "Invalid or expired confirmation code."
		template.ConfirmPage(username, msg, "").Render(r.Context(), w)
		return
	}

	template.ConfirmPage("", "", "Account confirmed! You can now sign in.").Render(r.Context(), w)
}
