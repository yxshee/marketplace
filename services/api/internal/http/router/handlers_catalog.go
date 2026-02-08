package router

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/auth"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/catalog"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/vendor"
)

type vendorCreateProductRequest struct {
	Title             string `json:"title"`
	Description       string `json:"description"`
	PriceInclTaxCents int64  `json:"price_incl_tax_cents"`
	Currency          string `json:"currency"`
}

type adminModerationRequest struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

func (a *api) handleVendorCreateProduct(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if identity.VendorID == nil {
		writeError(w, http.StatusBadRequest, "vendor profile required")
		return
	}

	registeredVendor, exists := a.vendorService.GetByID(*identity.VendorID)
	if !exists {
		writeError(w, http.StatusNotFound, "vendor not found")
		return
	}
	if registeredVendor.OwnerUserID != identity.UserID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req vendorCreateProductRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" || req.PriceInclTaxCents <= 0 || req.Currency == "" {
		writeError(w, http.StatusBadRequest, "title, currency and positive price are required")
		return
	}

	product := a.catalogService.CreateProduct(
		identity.UserID,
		registeredVendor.ID,
		req.Title,
		req.Description,
		req.Currency,
		req.PriceInclTaxCents,
	)

	writeJSON(w, http.StatusCreated, product)
}

func (a *api) handleVendorSubmitModeration(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if identity.VendorID == nil {
		writeError(w, http.StatusBadRequest, "vendor profile required")
		return
	}

	registeredVendor, exists := a.vendorService.GetByID(*identity.VendorID)
	if !exists {
		writeError(w, http.StatusNotFound, "vendor not found")
		return
	}
	if registeredVendor.VerificationState != vendor.VerificationVerified {
		writeError(w, http.StatusForbidden, "vendor must be verified before submission")
		return
	}

	productID := chi.URLParam(r, "productID")
	updatedProduct, err := a.catalogService.SubmitForModeration(productID, identity.UserID, registeredVendor.ID)
	if err != nil {
		switch {
		case errors.Is(err, catalog.ErrProductNotFound):
			writeError(w, http.StatusNotFound, "product not found")
		case errors.Is(err, catalog.ErrUnauthorizedProductAccess):
			writeError(w, http.StatusForbidden, "forbidden")
		case errors.Is(err, catalog.ErrInvalidStatusTransition):
			writeError(w, http.StatusConflict, "invalid product status transition")
		default:
			writeError(w, http.StatusBadRequest, "unable to submit moderation")
		}
		return
	}

	writeJSON(w, http.StatusOK, updatedProduct)
}

func (a *api) handleAdminModerateProduct(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	productID := chi.URLParam(r, "productID")
	if productID == "" {
		writeError(w, http.StatusBadRequest, "product id is required")
		return
	}

	var req adminModerationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updatedProduct, err := a.catalogService.ReviewProduct(
		productID,
		identity.UserID,
		catalog.ModerationDecision(req.Decision),
		req.Reason,
	)
	if err != nil {
		switch {
		case errors.Is(err, catalog.ErrProductNotFound):
			writeError(w, http.StatusNotFound, "product not found")
		case errors.Is(err, catalog.ErrInvalidModerationDecision):
			writeError(w, http.StatusBadRequest, "invalid moderation decision")
		case errors.Is(err, catalog.ErrInvalidStatusTransition):
			writeError(w, http.StatusConflict, "invalid product status transition")
		default:
			writeError(w, http.StatusBadRequest, "unable to moderate product")
		}
		return
	}

	writeJSON(w, http.StatusOK, updatedProduct)
}

func (a *api) handleCatalogList(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	vendorID := strings.TrimSpace(r.URL.Query().Get("vendor"))
	sortBy := catalog.SortOption(strings.TrimSpace(r.URL.Query().Get("sort")))
	limit := parseQueryInt(r, "limit", 20)
	offset := parseQueryInt(r, "offset", 0)
	priceMin := parseQueryInt64(r, "price_min", 0)
	priceMax := parseQueryInt64(r, "price_max", 0)
	minRating := parseQueryFloat64(r, "min_rating", 0)

	result := a.catalogService.Search(catalog.SearchParams{
		Query:     query,
		Category:  category,
		VendorID:  vendorID,
		PriceMin:  priceMin,
		PriceMax:  priceMax,
		MinRating: minRating,
		SortBy:    sortBy,
		Limit:     limit,
		Offset:    offset,
	}, func(vendorID string) bool {
		registeredVendor, exists := a.vendorService.GetByID(vendorID)
		if !exists {
			return false
		}
		return registeredVendor.VerificationState == vendor.VerificationVerified
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":  result.Items,
		"total":  result.Total,
		"limit":  limit,
		"offset": offset,
	})
}

func (a *api) handleCatalogProductDetail(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productID")
	product, exists := a.catalogService.GetProductByID(productID)
	if !exists || product.Status != catalog.ProductStatusApproved {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}

	registeredVendor, vendorExists := a.vendorService.GetByID(product.VendorID)
	if !vendorExists || registeredVendor.VerificationState != vendor.VerificationVerified {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"item": product,
		"vendor": map[string]string{
			"id":          registeredVendor.ID,
			"slug":        registeredVendor.Slug,
			"displayName": registeredVendor.DisplayName,
		},
	})
}

func (a *api) handleCatalogCategories(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": a.catalogService.ListCategories(),
	})
}

func parseQueryInt(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func parseQueryInt64(r *http.Request, key string, fallback int64) int64 {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return value
}

func parseQueryFloat64(r *http.Request, key string, fallback float64) float64 {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return value
}
