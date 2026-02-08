package router

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/auth"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/catalog"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/vendors"
)

type vendorCreateProductRequest struct {
	CategorySlug      string   `json:"category_slug"`
	Tags              []string `json:"tags"`
	StockQty          int32    `json:"stock_qty"`
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	PriceInclTaxCents int64    `json:"price_incl_tax_cents"`
	Currency          string   `json:"currency"`
}

type vendorUpdateProductRequest struct {
	CategorySlug      *string   `json:"category_slug"`
	Tags              *[]string `json:"tags"`
	StockQty          *int32    `json:"stock_qty"`
	Title             *string   `json:"title"`
	Description       *string   `json:"description"`
	PriceInclTaxCents *int64    `json:"price_incl_tax_cents"`
	Currency          *string   `json:"currency"`
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
	if req.StockQty < 0 {
		writeError(w, http.StatusBadRequest, "stock qty must be zero or positive")
		return
	}

	product := a.catalogService.CreateProductWithInput(catalog.CreateProductInput{
		OwnerUserID:       identity.UserID,
		VendorID:          registeredVendor.ID,
		Title:             req.Title,
		Description:       req.Description,
		CategorySlug:      req.CategorySlug,
		Tags:              req.Tags,
		PriceInclTaxCents: req.PriceInclTaxCents,
		Currency:          req.Currency,
		StockQty:          req.StockQty,
		Status:            catalog.ProductStatusDraft,
	})

	writeJSON(w, http.StatusCreated, product)
}

func (a *api) handleVendorListProducts(w http.ResponseWriter, r *http.Request) {
	identity, registeredVendor, ok := a.vendorOwnerContext(w, r)
	if !ok {
		return
	}

	items := a.catalogService.ListVendorProducts(identity.UserID, registeredVendor.ID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}

func (a *api) handleVendorUpdateProduct(w http.ResponseWriter, r *http.Request) {
	identity, registeredVendor, ok := a.vendorOwnerContext(w, r)
	if !ok {
		return
	}

	productID := chi.URLParam(r, "productID")
	if productID == "" {
		writeError(w, http.StatusBadRequest, "product id is required")
		return
	}

	var req vendorUpdateProductRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.CategorySlug == nil &&
		req.Tags == nil &&
		req.StockQty == nil &&
		req.Title == nil &&
		req.Description == nil &&
		req.PriceInclTaxCents == nil &&
		req.Currency == nil {
		writeError(w, http.StatusBadRequest, "at least one field is required")
		return
	}

	updated, err := a.catalogService.UpdateProduct(productID, identity.UserID, registeredVendor.ID, catalog.UpdateProductInput{
		CategorySlug:      req.CategorySlug,
		Tags:              req.Tags,
		StockQty:          req.StockQty,
		Title:             req.Title,
		Description:       req.Description,
		PriceInclTaxCents: req.PriceInclTaxCents,
		Currency:          req.Currency,
	})
	if err != nil {
		switch {
		case errors.Is(err, catalog.ErrProductNotFound):
			writeError(w, http.StatusNotFound, "product not found")
		case errors.Is(err, catalog.ErrUnauthorizedProductAccess):
			writeError(w, http.StatusForbidden, "forbidden")
		case errors.Is(err, catalog.ErrInvalidProductInput):
			writeError(w, http.StatusBadRequest, "invalid product payload")
		default:
			writeError(w, http.StatusBadRequest, "unable to update product")
		}
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (a *api) handleVendorDeleteProduct(w http.ResponseWriter, r *http.Request) {
	identity, registeredVendor, ok := a.vendorOwnerContext(w, r)
	if !ok {
		return
	}

	productID := chi.URLParam(r, "productID")
	if productID == "" {
		writeError(w, http.StatusBadRequest, "product id is required")
		return
	}

	if err := a.catalogService.DeleteProduct(productID, identity.UserID, registeredVendor.ID); err != nil {
		switch {
		case errors.Is(err, catalog.ErrProductNotFound):
			writeError(w, http.StatusNotFound, "product not found")
		case errors.Is(err, catalog.ErrUnauthorizedProductAccess):
			writeError(w, http.StatusForbidden, "forbidden")
		default:
			writeError(w, http.StatusBadRequest, "unable to delete product")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *api) handleVendorSubmitModeration(w http.ResponseWriter, r *http.Request) {
	identity, registeredVendor, ok := a.vendorOwnerContext(w, r)
	if !ok {
		return
	}
	if registeredVendor.VerificationState != vendors.VerificationVerified {
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

func (a *api) vendorOwnerContext(w http.ResponseWriter, r *http.Request) (auth.Identity, vendors.Vendor, bool) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return auth.Identity{}, vendors.Vendor{}, false
	}
	if identity.VendorID == nil {
		writeError(w, http.StatusBadRequest, "vendor profile required")
		return auth.Identity{}, vendors.Vendor{}, false
	}

	registeredVendor, exists := a.vendorService.GetByID(*identity.VendorID)
	if !exists {
		writeError(w, http.StatusNotFound, "vendor not found")
		return auth.Identity{}, vendors.Vendor{}, false
	}
	if registeredVendor.OwnerUserID != identity.UserID {
		writeError(w, http.StatusForbidden, "forbidden")
		return auth.Identity{}, vendors.Vendor{}, false
	}

	return identity, registeredVendor, true
}

func (a *api) handleAdminModerationList(w http.ResponseWriter, r *http.Request) {
	statusFilter := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("status")))
	targetStatus := catalog.ProductStatusPendingApproval
	if statusFilter != "" {
		switch catalog.ProductStatus(statusFilter) {
		case catalog.ProductStatusDraft,
			catalog.ProductStatusPendingApproval,
			catalog.ProductStatusApproved,
			catalog.ProductStatusRejected:
			targetStatus = catalog.ProductStatus(statusFilter)
		default:
			writeError(w, http.StatusBadRequest, "invalid moderation status filter")
			return
		}
	}

	items := a.catalogService.ListByStatus(targetStatus)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"total": len(items),
	})
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
		return registeredVendor.VerificationState == vendors.VerificationVerified
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
	if !vendorExists || registeredVendor.VerificationState != vendors.VerificationVerified {
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
