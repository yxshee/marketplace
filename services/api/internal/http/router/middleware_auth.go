package router

import (
	"net/http"
	"strings"

	"github.com/yxshee/marketplace-platform/services/api/internal/auth"
)

func (a *api) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, err := a.parseAccessIdentity(r.Header.Get("Authorization"))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		next.ServeHTTP(w, r.WithContext(auth.WithIdentity(r.Context(), *identity)))
	})
}

func (a *api) optionalAuthenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if header == "" {
			next.ServeHTTP(w, r)
			return
		}

		identity, err := a.parseAccessIdentity(header)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid access token")
			return
		}

		next.ServeHTTP(w, r.WithContext(auth.WithIdentity(r.Context(), *identity)))
	})
}

func (a *api) requirePermission(permission auth.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := auth.IdentityFromContext(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			if err := auth.MustBeAllowed(identity.Role, permission); err != nil {
				writeError(w, http.StatusForbidden, "forbidden")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (a *api) parseAccessIdentity(authorizationHeader string) (*auth.Identity, error) {
	token, err := bearerToken(authorizationHeader)
	if err != nil {
		return nil, err
	}

	claims, err := a.tokenManager.ParseAndValidate(token, auth.TokenTypeAccess)
	if err != nil {
		return nil, err
	}

	identity := &auth.Identity{
		UserID:    claims.UserID,
		Role:      claims.Role,
		SessionID: claims.SessionID,
		VendorID:  claims.VendorID,
	}
	return identity, nil
}
