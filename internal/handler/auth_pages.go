package handler

import (
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/c3d4r/app_scaffold/internal/auth"
	"github.com/c3d4r/app_scaffold/internal/template"
)

func (h *Handler) handleLoginPage(c echo.Context) error {
	if c.Request().Method == http.MethodGet {
		if h.cognitoClient == nil {
			return c.Redirect(http.StatusFound, "/auth/dev-sign-in")
		}
		return template.LoginPage("").Render(c.Request().Context(), c.Response().Writer)
	}

	if h.cognitoClient == nil {
		return c.String(http.StatusInternalServerError, "auth not configured")
	}

	username := strings.TrimSpace(c.FormValue("username"))
	password := c.FormValue("password")

	if username == "" || password == "" {
		return template.LoginPage("Username and password are required.").Render(c.Request().Context(), c.Response().Writer)
	}

	result, challenge, err := h.cognitoClient.SignIn(c.Request().Context(), username, password)
	if err != nil {
		log.Printf("sign in error for %q: %v", username, err)
		return template.LoginPage("Invalid username or password.").Render(c.Request().Context(), c.Response().Writer)
	}

	if challenge != nil {
		return template.LoginPage("Account requires a password change. Please use a different sign-in method.").Render(c.Request().Context(), c.Response().Writer)
	}

	claims, err := h.cognito.VerifyIDToken(c.Request().Context(), result.IDToken)
	if err != nil {
		log.Printf("verify id token: %v", err)
		return template.LoginPage("Failed to verify identity.").Render(c.Request().Context(), c.Response().Writer)
	}

	displayName := claims.Username
	if displayName == "" {
		displayName = claims.Email
	}

	session := auth.NewSession(claims.Sub, displayName, claims.Email)
	if err := h.sessionStore.PutSession(c.Request().Context(), session); err != nil {
		log.Printf("save session: %v", err)
		return template.LoginPage("Failed to create session.").Render(c.Request().Context(), c.Response().Writer)
	}

	c.SetCookie(&http.Cookie{
		Name:     "scaffold_session",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request().TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	dest := c.QueryParam("redirect")
	if dest == "" {
		dest = "/default"
	}
	return c.Redirect(http.StatusFound, dest)
}

func (h *Handler) handleSignupPage(c echo.Context) error {
	if c.Request().Method == http.MethodGet {
		if h.cognitoClient == nil {
			return c.String(http.StatusInternalServerError, "auth not configured")
		}
		return template.SignupPage("").Render(c.Request().Context(), c.Response().Writer)
	}

	if h.cognitoClient == nil {
		return c.String(http.StatusInternalServerError, "auth not configured")
	}

	username := strings.TrimSpace(c.FormValue("username"))
	email := strings.TrimSpace(c.FormValue("email"))
	password := c.FormValue("password")

	if username == "" || email == "" || password == "" {
		return template.SignupPage("All fields are required.").Render(c.Request().Context(), c.Response().Writer)
	}

	if len(password) < 8 {
		return template.SignupPage("Password must be at least 8 characters.").Render(c.Request().Context(), c.Response().Writer)
	}

	if err := h.cognitoClient.SignUp(c.Request().Context(), username, password, email); err != nil {
		log.Printf("sign up error for %q: %v", username, err)
		msg := "Sign up failed. Username may already exist."
		if strings.Contains(err.Error(), "UsernameExistsException") {
			msg = "An account with that username already exists."
		}
		return template.SignupPage(msg).Render(c.Request().Context(), c.Response().Writer)
	}

	return c.Redirect(http.StatusFound, "/auth/confirm?username="+username)
}

func (h *Handler) handleConfirmPage(c echo.Context) error {
	if h.cognitoClient == nil {
		return c.String(http.StatusInternalServerError, "auth not configured")
	}

	username := c.QueryParam("username")
	if c.Request().Method == http.MethodGet {
		if username == "" {
			return c.String(http.StatusBadRequest, "missing username")
		}
		return template.ConfirmPage(username, "", "").Render(c.Request().Context(), c.Response().Writer)
	}

	code := strings.TrimSpace(c.FormValue("code"))

	if username == "" || code == "" {
		return template.ConfirmPage(username, "Username and code are required.", "").Render(c.Request().Context(), c.Response().Writer)
	}

	if err := h.cognitoClient.ConfirmSignUp(c.Request().Context(), username, code); err != nil {
		log.Printf("confirm error for %q: %v", username, err)
		return template.ConfirmPage(username, "Invalid or expired confirmation code.", "").Render(c.Request().Context(), c.Response().Writer)
	}

	return template.ConfirmPage("", "", "Account confirmed! You can now sign in.").Render(c.Request().Context(), c.Response().Writer)
}
