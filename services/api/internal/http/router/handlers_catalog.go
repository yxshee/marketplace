package router

import (
	"errors"
	"net/http"

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

func (a *api) handleCatalogList(w http.ResponseWriter, _ *http.Request) {
	products := a.catalogService.ListVisibleProducts(func(vendorID string) bool {
		registeredVendor, exists := a.vendorService.GetByID(vendorID)
		if !exists {
			return false
		}
		return registeredVendor.VerificationState == vendor.VerificationVerified
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": products,
		"count": len(products),
	})
}
