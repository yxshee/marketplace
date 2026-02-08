package router

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/yxshee/marketplace-platform/services/api/internal/auth"
	"github.com/yxshee/marketplace-platform/services/api/internal/catalog"
	"github.com/yxshee/marketplace-platform/services/api/internal/commerce"
	"github.com/yxshee/marketplace-platform/services/api/internal/platform/identifier"
	"github.com/yxshee/marketplace-platform/services/api/internal/vendors"
)

const guestTokenHeader = "X-Guest-Token"

type cartAddItemRequest struct {
	ProductID string `json:"product_id"`
	Qty       int32  `json:"qty"`
}

type cartUpdateItemRequest struct {
	Qty int32 `json:"qty"`
}

type checkoutPlaceOrderRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
}

type cartResponse struct {
	commerce.Cart
	GuestToken string `json:"guest_token,omitempty"`
}

type checkoutQuoteResponse struct {
	commerce.CheckoutQuote
	GuestToken string `json:"guest_token,omitempty"`
}

type orderResponse struct {
	Order      commerce.Order `json:"order"`
	GuestToken string         `json:"guest_token,omitempty"`
}

func (a *api) handleCartGet(w http.ResponseWriter, r *http.Request) {
	actor, guestToken := checkoutActor(r)
	cart, err := a.commerce.GetCart(actor)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unable to resolve cart actor")
		return
	}

	writeBuyerResponse(w, http.StatusOK, cartResponse{
		Cart:       cart,
		GuestToken: guestToken,
	}, guestToken)
}

func (a *api) handleCartAddItem(w http.ResponseWriter, r *http.Request) {
	actor, guestToken := checkoutActor(r)

	var req cartAddItemRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Qty <= 0 {
		writeError(w, http.StatusBadRequest, "quantity must be positive")
		return
	}

	product, err := a.checkoutProduct(req.ProductID)
	if err != nil {
		switch {
		case errors.Is(err, catalog.ErrProductNotFound):
			writeError(w, http.StatusNotFound, "product not found")
		default:
			writeError(w, http.StatusConflict, "product unavailable")
		}
		return
	}
	if product.StockQty <= 0 {
		writeError(w, http.StatusConflict, "product out of stock")
		return
	}

	cart, err := a.commerce.UpsertItem(actor, commerce.ProductSnapshot{
		ID:                    product.ID,
		VendorID:              product.VendorID,
		Title:                 product.Title,
		Currency:              product.Currency,
		UnitPriceInclTaxCents: product.PriceInclTaxCents,
		StockQty:              product.StockQty,
	}, req.Qty)
	if err != nil {
		a.writeCartError(w, err)
		return
	}

	writeBuyerResponse(w, http.StatusOK, cartResponse{Cart: cart, GuestToken: guestToken}, guestToken)
}

func (a *api) handleCartUpdateItem(w http.ResponseWriter, r *http.Request) {
	actor, guestToken := checkoutActor(r)
	itemID := chi.URLParam(r, "itemID")
	if strings.TrimSpace(itemID) == "" {
		writeError(w, http.StatusBadRequest, "item id is required")
		return
	}

	var req cartUpdateItemRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Qty <= 0 {
		writeError(w, http.StatusBadRequest, "quantity must be positive")
		return
	}

	cart, err := a.commerce.UpdateItemQty(actor, itemID, req.Qty)
	if err != nil {
		a.writeCartError(w, err)
		return
	}

	writeBuyerResponse(w, http.StatusOK, cartResponse{Cart: cart, GuestToken: guestToken}, guestToken)
}

func (a *api) handleCartDeleteItem(w http.ResponseWriter, r *http.Request) {
	actor, guestToken := checkoutActor(r)
	itemID := chi.URLParam(r, "itemID")
	if strings.TrimSpace(itemID) == "" {
		writeError(w, http.StatusBadRequest, "item id is required")
		return
	}

	cart, err := a.commerce.RemoveItem(actor, itemID)
	if err != nil {
		a.writeCartError(w, err)
		return
	}

	writeBuyerResponse(w, http.StatusOK, cartResponse{Cart: cart, GuestToken: guestToken}, guestToken)
}

func (a *api) handleCheckoutQuote(w http.ResponseWriter, r *http.Request) {
	actor, guestToken := checkoutActor(r)

	quote, err := a.commerce.Quote(actor)
	if err != nil {
		if errors.Is(err, commerce.ErrCartEmpty) {
			writeError(w, http.StatusConflict, "cart is empty")
			return
		}
		writeError(w, http.StatusBadRequest, "unable to prepare checkout quote")
		return
	}

	writeBuyerResponse(w, http.StatusOK, checkoutQuoteResponse{
		CheckoutQuote: quote,
		GuestToken:    guestToken,
	}, guestToken)
}

func (a *api) handleCheckoutPlaceOrder(w http.ResponseWriter, r *http.Request) {
	actor, guestToken := checkoutActor(r)

	var req checkoutPlaceOrderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := a.commerce.PlaceOrder(actor, req.IdempotencyKey)
	if err != nil {
		switch {
		case errors.Is(err, commerce.ErrCartEmpty):
			writeError(w, http.StatusConflict, "cart is empty")
		case errors.Is(err, commerce.ErrIdempotencyKey):
			writeError(w, http.StatusBadRequest, "idempotency key is required")
		default:
			writeError(w, http.StatusBadRequest, "unable to place order")
		}
		return
	}

	writeBuyerResponse(w, http.StatusCreated, orderResponse{
		Order:      order,
		GuestToken: guestToken,
	}, guestToken)
}

func (a *api) handleOrderByID(w http.ResponseWriter, r *http.Request) {
	actor, guestToken := checkoutActor(r)
	orderID := chi.URLParam(r, "orderID")
	if strings.TrimSpace(orderID) == "" {
		writeError(w, http.StatusBadRequest, "order id is required")
		return
	}

	order, found, err := a.commerce.GetOrder(actor, orderID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unable to resolve order actor")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}

	writeBuyerResponse(w, http.StatusOK, orderResponse{Order: order, GuestToken: guestToken}, guestToken)
}

func (a *api) checkoutProduct(productID string) (catalog.Product, error) {
	product, exists := a.catalogService.GetProductByID(strings.TrimSpace(productID))
	if !exists || product.Status != catalog.ProductStatusApproved {
		return catalog.Product{}, catalog.ErrProductNotFound
	}

	registeredVendor, exists := a.vendorService.GetByID(product.VendorID)
	if !exists || registeredVendor.VerificationState != vendors.VerificationVerified {
		return catalog.Product{}, catalog.ErrProductNotFound
	}

	return product, nil
}

func (a *api) writeCartError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, commerce.ErrCartItemNotFound):
		writeError(w, http.StatusNotFound, "cart item not found")
	case errors.Is(err, commerce.ErrInsufficientStock):
		writeError(w, http.StatusConflict, "insufficient stock")
	case errors.Is(err, commerce.ErrCurrencyMismatch):
		writeError(w, http.StatusConflict, "currency mismatch")
	case errors.Is(err, commerce.ErrInvalidQuantity), errors.Is(err, commerce.ErrInvalidProduct), errors.Is(err, commerce.ErrInvalidActor):
		writeError(w, http.StatusBadRequest, "invalid cart request")
	default:
		writeError(w, http.StatusBadRequest, "unable to update cart")
	}
}

func checkoutActor(r *http.Request) (commerce.Actor, string) {
	if identity, ok := auth.IdentityFromContext(r.Context()); ok {
		return commerce.Actor{BuyerUserID: identity.UserID}, ""
	}

	guestToken := strings.TrimSpace(r.Header.Get(guestTokenHeader))
	if guestToken == "" {
		guestToken = identifier.New("gst")
	}
	return commerce.Actor{GuestToken: guestToken}, guestToken
}

func writeBuyerResponse(w http.ResponseWriter, statusCode int, payload interface{}, guestToken string) {
	if strings.TrimSpace(guestToken) != "" {
		w.Header().Set(guestTokenHeader, guestToken)
	}
	writeJSON(w, statusCode, payload)
}
