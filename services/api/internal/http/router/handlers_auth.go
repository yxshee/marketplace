package router

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/auth"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/platform/identifier"
)

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type authResponse struct {
	AccessToken      string      `json:"access_token"`
	RefreshToken     string      `json:"refresh_token"`
	AccessExpiresAt  time.Time   `json:"access_expires_at"`
	RefreshExpiresAt time.Time   `json:"refresh_expires_at"`
	User             authUserDTO `json:"user"`
}

type authUserDTO struct {
	ID       string     `json:"id"`
	Email    string     `json:"email"`
	Role     auth.Role  `json:"role"`
	VendorID *string    `json:"vendor_id,omitempty"`
	Created  *time.Time `json:"created_at,omitempty"`
}

func toAuthUserDTO(user auth.User) authUserDTO {
	created := user.CreatedAt
	return authUserDTO{
		ID:       user.ID,
		Email:    user.Email,
		Role:     user.Role,
		VendorID: user.VendorID,
		Created:  &created,
	}
}

func (a *api) issueTokensForUser(user auth.User) (authResponse, error) {
	sessionID := identifier.New("ses")
	pair, err := a.tokenManager.IssueTokenPair(user, sessionID)
	if err != nil {
		return authResponse{}, err
	}

	a.authService.SaveSession(auth.Session{
		ID:               sessionID,
		UserID:           user.ID,
		RefreshTokenHash: auth.HashToken(pair.RefreshToken),
		ExpiresAt:        pair.RefreshExpiresAt,
	})

	return authResponse{
		AccessToken:      pair.AccessToken,
		RefreshToken:     pair.RefreshToken,
		AccessExpiresAt:  pair.AccessExpiresAt,
		RefreshExpiresAt: pair.RefreshExpiresAt,
		User:             toAuthUserDTO(user),
	}, nil
}

func (a *api) handleAuthRegister(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := a.authService.Register(req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailInUse):
			writeError(w, http.StatusConflict, "email already registered")
		case errors.Is(err, auth.ErrWeakPassword):
			writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		default:
			writeError(w, http.StatusBadRequest, "registration failed")
		}
		return
	}

	response, err := a.issueTokensForUser(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token issuance failed")
		return
	}

	writeJSON(w, http.StatusCreated, response)
}

func (a *api) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := a.authService.Authenticate(req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	response, err := a.issueTokensForUser(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token issuance failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (a *api) handleAuthRefresh(w http.ResponseWriter, r *http.Request) {
	var req authRefreshRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	claims, err := a.tokenManager.ParseAndValidate(strings.TrimSpace(req.RefreshToken), auth.TokenTypeRefresh)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	session, exists := a.authService.GetSession(claims.SessionID)
	if !exists || session.UserID != claims.UserID {
		writeError(w, http.StatusUnauthorized, "invalid refresh session")
		return
	}
	if session.ExpiresAt.Before(time.Now().UTC()) {
		a.authService.DeleteSession(session.ID)
		writeError(w, http.StatusUnauthorized, "refresh session expired")
		return
	}
	if session.RefreshTokenHash != auth.HashToken(req.RefreshToken) {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	user, exists := a.authService.GetUserByID(claims.UserID)
	if !exists {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	a.authService.DeleteSession(session.ID)

	response, err := a.issueTokensForUser(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token issuance failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (a *api) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req authRefreshRequest
	if err := decodeJSON(r, &req); err == nil && strings.TrimSpace(req.RefreshToken) != "" {
		if claims, parseErr := a.tokenManager.ParseAndValidate(strings.TrimSpace(req.RefreshToken), auth.TokenTypeRefresh); parseErr == nil {
			a.authService.DeleteSession(claims.SessionID)
			writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
			return
		}
	}

	a.authService.DeleteSession(identity.SessionID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (a *api) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	user, exists := a.authService.GetUserByID(identity.UserID)
	if !exists {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, toAuthUserDTO(user))
}
