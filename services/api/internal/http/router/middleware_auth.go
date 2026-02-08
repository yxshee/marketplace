package router

import (
	"net/http"

	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/auth"
)

func (a *api) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := bearerToken(r.Header.Get("Authorization"))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		claims, err := a.tokenManager.ParseAndValidate(token, auth.TokenTypeAccess)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid access token")
			return
		}

		identity := auth.Identity{
			UserID:    claims.UserID,
			Role:      claims.Role,
			SessionID: claims.SessionID,
			VendorID:  claims.VendorID,
		}

		next.ServeHTTP(w, r.WithContext(auth.WithIdentity(r.Context(), identity)))
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
