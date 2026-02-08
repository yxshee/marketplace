package router

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/auth"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/catalog"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/config"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/vendor"
)

type healthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

type api struct {
	authService    *auth.Service
	tokenManager   *auth.TokenManager
	vendorService  *vendor.Service
	catalogService *catalog.Service
	defaultCommBPS int32
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(healthResponse{
		Status:    "ok",
		Service:   "marketplace-api",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// New creates a production-ready chi router with baseline middleware and routes.
func New(cfg config.Config) (http.Handler, error) {
	tokenManager, err := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	if err != nil {
		return nil, err
	}

	authService := auth.NewService(auth.BuildBootstrapRoleMap(
		cfg.SuperAdminEmails,
		cfg.SupportEmails,
		cfg.FinanceEmails,
		cfg.CatalogModEmails,
	))

	apiHandlers := &api{
		authService:    authService,
		tokenManager:   tokenManager,
		vendorService:  vendor.NewService(),
		catalogService: catalog.NewService(),
		defaultCommBPS: cfg.DefaultCommission,
	}
	if cfg.Environment == "development" {
		apiHandlers.seedDevelopmentCatalog()
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", healthHandler)

	r.Route("/api/v1", func(v1 chi.Router) {
		v1.Get("/health", healthHandler)
		v1.Get("/catalog/categories", apiHandlers.handleCatalogCategories)
		v1.Get("/catalog/products", apiHandlers.handleCatalogList)
		v1.Get("/catalog/products/{productID}", apiHandlers.handleCatalogProductDetail)
		v1.Post("/auth/register", apiHandlers.handleAuthRegister)
		v1.Post("/auth/login", apiHandlers.handleAuthLogin)
		v1.Post("/auth/refresh", apiHandlers.handleAuthRefresh)

		v1.Group(func(private chi.Router) {
			private.Use(apiHandlers.authenticate)
			private.Get("/auth/me", apiHandlers.handleAuthMe)
			private.Post("/auth/logout", apiHandlers.handleAuthLogout)

			private.Post("/vendors/register", apiHandlers.handleVendorRegister)
			private.Get("/vendor/verification-status", apiHandlers.handleVendorVerificationStatus)

			private.Group(func(vendorRoutes chi.Router) {
				vendorRoutes.Use(apiHandlers.requirePermission(auth.PermissionManageVendorProducts))
				vendorRoutes.Post("/vendor/products", apiHandlers.handleVendorCreateProduct)
				vendorRoutes.Post("/vendor/products/{productID}/submit-moderation", apiHandlers.handleVendorSubmitModeration)
			})

			private.Group(func(adminRoutes chi.Router) {
				adminRoutes.Use(apiHandlers.requirePermission(auth.PermissionManageVendorVerification))
				adminRoutes.Patch("/admin/vendors/{vendorID}/verification", apiHandlers.handleAdminVendorVerification)
			})

			private.Group(func(adminRoutes chi.Router) {
				adminRoutes.Use(apiHandlers.requirePermission(auth.PermissionManageCommission))
				adminRoutes.Patch("/admin/vendors/{vendorID}/commission", apiHandlers.handleAdminVendorCommission)
			})

			private.Group(func(adminRoutes chi.Router) {
				adminRoutes.Use(apiHandlers.requirePermission(auth.PermissionModerateProducts))
				adminRoutes.Patch("/admin/moderation/products/{productID}", apiHandlers.handleAdminModerateProduct)
			})
		})
	})

	return r, nil
}
