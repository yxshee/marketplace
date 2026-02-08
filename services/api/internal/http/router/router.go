package router

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/auditlog"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/auth"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/catalog"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/commerce"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/config"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/coupons"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/invoices"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/payments"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/promotions"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/refunds"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/vendors"
)

type healthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

type api struct {
	authService    *auth.Service
	tokenManager   *auth.TokenManager
	vendorService  *vendors.Service
	catalogService *catalog.Service
	coupons        *coupons.Service
	promotions     *promotions.Service
	auditLogs      *auditlog.Service
	commerce       *commerce.Service
	invoices       *invoices.Service
	payments       *payments.Service
	refunds        *refunds.Service
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

	var stripeClient payments.StripeClient = payments.NewMockStripeClient()
	if strings.EqualFold(strings.TrimSpace(cfg.StripeMode), "live") {
		stripeClient = payments.NewLiveStripeClient(cfg.StripeSecretKey)
	}

	commerceService := commerce.NewService(500)
	apiHandlers := &api{
		authService:    authService,
		tokenManager:   tokenManager,
		vendorService:  vendors.NewService(),
		catalogService: catalog.NewService(),
		coupons:        coupons.NewService(),
		promotions:     promotions.NewService(),
		auditLogs:      auditlog.NewService(),
		commerce:       commerceService,
		invoices: invoices.NewService(invoices.Config{
			PlatformName:         "Marketplace Gumroad Inspired",
			PlatformLegalEntity:  "Marketplace Gumroad Inspired LLC",
			PlatformSupportEmail: "support@marketplace.local",
			PlatformAddress:      "Global operations",
		}),
		defaultCommBPS: cfg.DefaultCommission,
		payments: payments.NewService(payments.Config{
			WebhookSecret: cfg.StripeWebhookSecret,
			StripeClient:  stripeClient,
			MarkOrderPaid: func(orderID string) bool {
				_, ok := commerceService.MarkOrderPaid(orderID)
				return ok
			},
			MarkOrderPaymentFailed: func(orderID string) bool {
				_, ok := commerceService.MarkOrderPaymentFailed(orderID)
				return ok
			},
			MarkOrderCODConfirmed: func(orderID string) bool {
				_, ok := commerceService.MarkOrderCODConfirmed(orderID)
				return ok
			},
		}),
		refunds: refunds.NewService(),
	}
	if cfg.Environment == "development" {
		apiHandlers.seedDevelopmentCatalog()
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	maxBodyBytes := cfg.MaxRequestBodyBytes
	if maxBodyBytes <= 0 {
		maxBodyBytes = 1 << 20
	}
	r.Use(middleware.RequestSize(maxBodyBytes))
	r.Use(securityHeaders)
	if cfg.EnableRateLimit {
		globalLimiter := newRequestRateLimiter(cfg.GlobalRateLimitRPS, cfg.GlobalRateLimitBurst, 2*time.Minute)
		r.Use(globalLimiter.middleware)
	}

	authRateLimitMiddleware := func(next http.Handler) http.Handler { return next }
	if cfg.EnableRateLimit {
		authLimiter := newRequestRateLimiter(cfg.AuthRateLimitRPS, cfg.AuthRateLimitBurst, 10*time.Minute)
		authRateLimitMiddleware = authLimiter.middleware
	}

	r.Get("/health", healthHandler)

	r.Route("/api/v1", func(v1 chi.Router) {
		v1.Get("/health", healthHandler)
		v1.Get("/catalog/categories", apiHandlers.handleCatalogCategories)
		v1.Get("/catalog/products", apiHandlers.handleCatalogList)
		v1.Get("/catalog/products/{productID}", apiHandlers.handleCatalogProductDetail)
		v1.Post("/webhooks/stripe", apiHandlers.handleStripeWebhook)

		v1.Group(func(buyerFlow chi.Router) {
			buyerFlow.Use(apiHandlers.optionalAuthenticate)
			buyerFlow.Get("/cart", apiHandlers.handleCartGet)
			buyerFlow.Post("/cart/items", apiHandlers.handleCartAddItem)
			buyerFlow.Patch("/cart/items/{itemID}", apiHandlers.handleCartUpdateItem)
			buyerFlow.Delete("/cart/items/{itemID}", apiHandlers.handleCartDeleteItem)
			buyerFlow.Post("/checkout/quote", apiHandlers.handleCheckoutQuote)
			buyerFlow.Post("/checkout/place-order", apiHandlers.handleCheckoutPlaceOrder)
			buyerFlow.Get("/payments/settings", apiHandlers.handleBuyerPaymentSettingsGet)
			buyerFlow.Post("/payments/stripe/intent", apiHandlers.handleStripeCreateIntent)
			buyerFlow.Post("/payments/cod/confirm", apiHandlers.handleCODConfirmPayment)
			buyerFlow.Get("/orders/{orderID}", apiHandlers.handleOrderByID)
			buyerFlow.Post("/orders/{orderID}/refund-requests", apiHandlers.handleBuyerCreateRefundRequest)
			buyerFlow.Get("/invoices/{orderID}/download", apiHandlers.handleInvoiceDownload)
		})

		v1.With(authRateLimitMiddleware).Post("/auth/register", apiHandlers.handleAuthRegister)
		v1.With(authRateLimitMiddleware).Post("/auth/login", apiHandlers.handleAuthLogin)
		v1.With(authRateLimitMiddleware).Post("/auth/refresh", apiHandlers.handleAuthRefresh)

		v1.Group(func(private chi.Router) {
			private.Use(apiHandlers.authenticate)
			private.Get("/auth/me", apiHandlers.handleAuthMe)
			private.Post("/auth/logout", apiHandlers.handleAuthLogout)

			private.Post("/vendors/register", apiHandlers.handleVendorRegister)
			private.Get("/vendor/profile", apiHandlers.handleVendorVerificationStatus)
			private.Get("/vendor/verification-status", apiHandlers.handleVendorVerificationStatus)

			private.Group(func(vendorRoutes chi.Router) {
				vendorRoutes.Use(apiHandlers.requirePermission(auth.PermissionManageVendorProducts))
				vendorRoutes.Get("/vendor/products", apiHandlers.handleVendorListProducts)
				vendorRoutes.Post("/vendor/products", apiHandlers.handleVendorCreateProduct)
				vendorRoutes.Patch("/vendor/products/{productID}", apiHandlers.handleVendorUpdateProduct)
				vendorRoutes.Delete("/vendor/products/{productID}", apiHandlers.handleVendorDeleteProduct)
				vendorRoutes.Post("/vendor/products/{productID}/submit-moderation", apiHandlers.handleVendorSubmitModeration)
			})

			private.Group(func(vendorRoutes chi.Router) {
				vendorRoutes.Use(apiHandlers.requirePermission(auth.PermissionManageVendorCoupons))
				vendorRoutes.Get("/vendor/coupons", apiHandlers.handleVendorListCoupons)
				vendorRoutes.Post("/vendor/coupons", apiHandlers.handleVendorCreateCoupon)
				vendorRoutes.Patch("/vendor/coupons/{couponID}", apiHandlers.handleVendorUpdateCoupon)
				vendorRoutes.Delete("/vendor/coupons/{couponID}", apiHandlers.handleVendorDeleteCoupon)
			})

			private.Group(func(vendorRoutes chi.Router) {
				vendorRoutes.Use(apiHandlers.requirePermission(auth.PermissionManageShipmentOrders))
				vendorRoutes.Get("/vendor/shipments", apiHandlers.handleVendorListShipments)
				vendorRoutes.Get("/vendor/shipments/{shipmentID}", apiHandlers.handleVendorShipmentDetail)
				vendorRoutes.Patch("/vendor/shipments/{shipmentID}/status", apiHandlers.handleVendorShipmentStatusUpdate)
			})

			private.Group(func(vendorRoutes chi.Router) {
				vendorRoutes.Use(apiHandlers.requirePermission(auth.PermissionManageRefundDecisions))
				vendorRoutes.Get("/vendor/refund-requests", apiHandlers.handleVendorListRefundRequests)
				vendorRoutes.Patch("/vendor/refund-requests/{refundRequestID}/decision", apiHandlers.handleVendorRefundDecision)
			})

			private.Group(func(vendorRoutes chi.Router) {
				vendorRoutes.Use(apiHandlers.requirePermission(auth.PermissionViewVendorAnalytics))
				vendorRoutes.Get("/vendor/analytics/overview", apiHandlers.handleVendorAnalyticsOverview)
				vendorRoutes.Get("/vendor/analytics/top-products", apiHandlers.handleVendorAnalyticsTopProducts)
				vendorRoutes.Get("/vendor/analytics/coupons", apiHandlers.handleVendorAnalyticsCoupons)
			})

			private.Group(func(adminRoutes chi.Router) {
				adminRoutes.Use(apiHandlers.requirePermission(auth.PermissionManageVendorVerification))
				adminRoutes.Get("/admin/vendors", apiHandlers.handleAdminVendorList)
				adminRoutes.Patch("/admin/vendors/{vendorID}/verification", apiHandlers.handleAdminVendorVerification)
			})

			private.Group(func(adminRoutes chi.Router) {
				adminRoutes.Use(apiHandlers.requirePermission(auth.PermissionManageCommission))
				adminRoutes.Patch("/admin/vendors/{vendorID}/commission", apiHandlers.handleAdminVendorCommission)
			})

			private.Group(func(adminRoutes chi.Router) {
				adminRoutes.Use(apiHandlers.requirePermission(auth.PermissionModerateProducts))
				adminRoutes.Get("/admin/moderation/products", apiHandlers.handleAdminModerationList)
				adminRoutes.Patch("/admin/moderation/products/{productID}", apiHandlers.handleAdminModerateProduct)
			})

			private.Group(func(adminRoutes chi.Router) {
				adminRoutes.Use(apiHandlers.requirePermission(auth.PermissionManageOrdersOperations))
				adminRoutes.Get("/admin/orders", apiHandlers.handleAdminOrdersList)
				adminRoutes.Get("/admin/orders/{orderID}", apiHandlers.handleAdminOrderDetail)
				adminRoutes.Patch("/admin/orders/{orderID}/status", apiHandlers.handleAdminOrderStatusUpdate)
			})

			private.Group(func(adminRoutes chi.Router) {
				adminRoutes.Use(apiHandlers.requirePermission(auth.PermissionManagePromotions))
				adminRoutes.Get("/admin/promotions", apiHandlers.handleAdminPromotionsList)
				adminRoutes.Post("/admin/promotions", apiHandlers.handleAdminPromotionCreate)
				adminRoutes.Patch("/admin/promotions/{promotionID}", apiHandlers.handleAdminPromotionUpdate)
				adminRoutes.Delete("/admin/promotions/{promotionID}", apiHandlers.handleAdminPromotionDelete)
			})

			private.Group(func(adminRoutes chi.Router) {
				adminRoutes.Use(apiHandlers.requirePermission(auth.PermissionViewAuditLogs))
				adminRoutes.Get("/admin/audit-logs", apiHandlers.handleAdminAuditLogsList)
			})

			private.Group(func(adminRoutes chi.Router) {
				adminRoutes.Use(apiHandlers.requirePermission(auth.PermissionViewAdminAnalytics))
				adminRoutes.Get("/admin/dashboard/overview", apiHandlers.handleAdminDashboardOverview)
				adminRoutes.Get("/admin/analytics/revenue", apiHandlers.handleAdminAnalyticsRevenue)
				adminRoutes.Get("/admin/analytics/vendors", apiHandlers.handleAdminAnalyticsVendors)
			})

			private.Group(func(adminRoutes chi.Router) {
				adminRoutes.Use(apiHandlers.requirePermission(auth.PermissionManagePaymentSettings))
				adminRoutes.Get("/admin/settings/payments", apiHandlers.handleAdminPaymentSettingsGet)
				adminRoutes.Patch("/admin/settings/payments", apiHandlers.handleAdminPaymentSettingsPatch)
			})
		})
	})

	return r, nil
}
