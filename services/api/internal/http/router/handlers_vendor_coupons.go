package router

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/yxshee/marketplace-platform/services/api/internal/coupons"
)

type vendorCreateCouponRequest struct {
	Code          string  `json:"code"`
	DiscountType  string  `json:"discount_type"`
	DiscountValue int64   `json:"discount_value"`
	StartsAt      *string `json:"starts_at,omitempty"`
	EndsAt        *string `json:"ends_at,omitempty"`
	UsageLimit    *int32  `json:"usage_limit,omitempty"`
	Active        *bool   `json:"active,omitempty"`
}

type vendorUpdateCouponRequest struct {
	Code          *string `json:"code,omitempty"`
	DiscountType  *string `json:"discount_type,omitempty"`
	DiscountValue *int64  `json:"discount_value,omitempty"`
	Active        *bool   `json:"active,omitempty"`
}

func (a *api) handleVendorListCoupons(w http.ResponseWriter, r *http.Request) {
	_, registeredVendor, ok := a.vendorOwnerContext(w, r)
	if !ok {
		return
	}

	items := a.coupons.ListByVendor(registeredVendor.ID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}

func (a *api) handleVendorCreateCoupon(w http.ResponseWriter, r *http.Request) {
	_, registeredVendor, ok := a.vendorOwnerContext(w, r)
	if !ok {
		return
	}

	var req vendorCreateCouponRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	startsAt, endsAt, err := parseCouponWindow(req.StartsAt, req.EndsAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid coupon date window")
		return
	}

	created, err := a.coupons.Create(registeredVendor.ID, coupons.CreateCouponInput{
		Code:          req.Code,
		DiscountType:  coupons.DiscountType(strings.TrimSpace(req.DiscountType)),
		DiscountValue: req.DiscountValue,
		StartsAt:      startsAt,
		EndsAt:        endsAt,
		UsageLimit:    req.UsageLimit,
		Active:        req.Active,
	})
	if err != nil {
		switch {
		case errors.Is(err, coupons.ErrCouponCodeInUse):
			writeError(w, http.StatusConflict, "coupon code already exists")
		case errors.Is(err, coupons.ErrInvalidCouponInput):
			writeError(w, http.StatusBadRequest, "invalid coupon payload")
		default:
			writeError(w, http.StatusBadRequest, "unable to create coupon")
		}
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (a *api) handleVendorUpdateCoupon(w http.ResponseWriter, r *http.Request) {
	_, registeredVendor, ok := a.vendorOwnerContext(w, r)
	if !ok {
		return
	}

	couponID := chi.URLParam(r, "couponID")
	if couponID == "" {
		writeError(w, http.StatusBadRequest, "coupon id is required")
		return
	}

	var req vendorUpdateCouponRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Code == nil && req.DiscountType == nil && req.DiscountValue == nil && req.Active == nil {
		writeError(w, http.StatusBadRequest, "at least one field is required")
		return
	}

	var discountType *coupons.DiscountType
	if req.DiscountType != nil {
		value := coupons.DiscountType(strings.TrimSpace(*req.DiscountType))
		discountType = &value
	}

	updated, err := a.coupons.Update(registeredVendor.ID, couponID, coupons.UpdateCouponInput{
		Code:          req.Code,
		DiscountType:  discountType,
		DiscountValue: req.DiscountValue,
		Active:        req.Active,
	})
	if err != nil {
		switch {
		case errors.Is(err, coupons.ErrCouponNotFound):
			writeError(w, http.StatusNotFound, "coupon not found")
		case errors.Is(err, coupons.ErrCouponCodeInUse):
			writeError(w, http.StatusConflict, "coupon code already exists")
		case errors.Is(err, coupons.ErrUnauthorizedCouponScope):
			writeError(w, http.StatusForbidden, "forbidden")
		case errors.Is(err, coupons.ErrInvalidCouponInput):
			writeError(w, http.StatusBadRequest, "invalid coupon payload")
		default:
			writeError(w, http.StatusBadRequest, "unable to update coupon")
		}
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (a *api) handleVendorDeleteCoupon(w http.ResponseWriter, r *http.Request) {
	_, registeredVendor, ok := a.vendorOwnerContext(w, r)
	if !ok {
		return
	}

	couponID := chi.URLParam(r, "couponID")
	if couponID == "" {
		writeError(w, http.StatusBadRequest, "coupon id is required")
		return
	}

	if err := a.coupons.Delete(registeredVendor.ID, couponID); err != nil {
		switch {
		case errors.Is(err, coupons.ErrCouponNotFound):
			writeError(w, http.StatusNotFound, "coupon not found")
		case errors.Is(err, coupons.ErrUnauthorizedCouponScope):
			writeError(w, http.StatusForbidden, "forbidden")
		default:
			writeError(w, http.StatusBadRequest, "unable to delete coupon")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseCouponWindow(startsAtRaw, endsAtRaw *string) (*time.Time, *time.Time, error) {
	parseOne := func(raw *string) (*time.Time, error) {
		if raw == nil || strings.TrimSpace(*raw) == "" {
			return nil, nil
		}
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*raw))
		if err != nil {
			return nil, err
		}
		utc := parsed.UTC()
		return &utc, nil
	}

	startsAt, err := parseOne(startsAtRaw)
	if err != nil {
		return nil, nil, err
	}
	endsAt, err := parseOne(endsAtRaw)
	if err != nil {
		return nil, nil, err
	}

	return startsAt, endsAt, nil
}
