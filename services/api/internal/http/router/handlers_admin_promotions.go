package router

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/promotions"
)

type adminPromotionListResponse struct {
	Items  []promotions.Promotion `json:"items"`
	Total  int                    `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}

type adminPromotionCreateRequest struct {
	Name      string          `json:"name"`
	RuleJSON  json.RawMessage `json:"rule_json"`
	StartsAt  *time.Time      `json:"starts_at"`
	EndsAt    *time.Time      `json:"ends_at"`
	Stackable *bool           `json:"stackable"`
	Active    *bool           `json:"active"`
}

type adminPromotionUpdateRequest struct {
	Name      *string          `json:"name"`
	RuleJSON  *json.RawMessage `json:"rule_json"`
	StartsAt  *time.Time       `json:"starts_at"`
	EndsAt    *time.Time       `json:"ends_at"`
	Stackable *bool            `json:"stackable"`
	Active    *bool            `json:"active"`
}

func (a *api) handleAdminPromotionsList(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parsePagination(r, 50, 200)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	items := a.promotions.List()
	total := len(items)
	start, end := paginate(total, limit, offset)
	writeJSON(w, http.StatusOK, adminPromotionListResponse{
		Items:  items[start:end],
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (a *api) handleAdminPromotionCreate(w http.ResponseWriter, r *http.Request) {
	var req adminPromotionCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	promotion, err := a.promotions.Create(promotions.CreatePromotionInput{
		Name:      req.Name,
		RuleJSON:  req.RuleJSON,
		StartsAt:  req.StartsAt,
		EndsAt:    req.EndsAt,
		Stackable: req.Stackable,
		Active:    req.Active,
	})
	if err != nil {
		switch {
		case errors.Is(err, promotions.ErrInvalidPromotion):
			writeError(w, http.StatusBadRequest, "invalid promotion payload")
		default:
			writeError(w, http.StatusInternalServerError, "unable to create promotion")
		}
		return
	}

	a.recordAuditLog(
		r,
		"promotion_created",
		"promotion",
		promotion.ID,
		nil,
		promotion,
		nil,
	)
	writeJSON(w, http.StatusCreated, promotion)
}

func (a *api) handleAdminPromotionUpdate(w http.ResponseWriter, r *http.Request) {
	promotionID := strings.TrimSpace(chi.URLParam(r, "promotionID"))
	if promotionID == "" {
		writeError(w, http.StatusBadRequest, "promotion id is required")
		return
	}
	previousPromotion, exists := a.promotions.GetByID(promotionID)
	if !exists {
		writeError(w, http.StatusNotFound, "promotion not found")
		return
	}

	var req adminPromotionUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	promotion, err := a.promotions.Update(promotionID, promotions.UpdatePromotionInput{
		Name:      req.Name,
		RuleJSON:  req.RuleJSON,
		StartsAt:  req.StartsAt,
		EndsAt:    req.EndsAt,
		Stackable: req.Stackable,
		Active:    req.Active,
	})
	if err != nil {
		switch {
		case errors.Is(err, promotions.ErrNoPromotionChanges),
			errors.Is(err, promotions.ErrInvalidPromotion):
			writeError(w, http.StatusBadRequest, "invalid promotion payload")
		case errors.Is(err, promotions.ErrPromotionNotFound):
			writeError(w, http.StatusNotFound, "promotion not found")
		default:
			writeError(w, http.StatusInternalServerError, "unable to update promotion")
		}
		return
	}

	a.recordAuditLog(
		r,
		"promotion_updated",
		"promotion",
		promotion.ID,
		previousPromotion,
		promotion,
		nil,
	)
	writeJSON(w, http.StatusOK, promotion)
}

func (a *api) handleAdminPromotionDelete(w http.ResponseWriter, r *http.Request) {
	promotionID := strings.TrimSpace(chi.URLParam(r, "promotionID"))
	if promotionID == "" {
		writeError(w, http.StatusBadRequest, "promotion id is required")
		return
	}
	previousPromotion, exists := a.promotions.GetByID(promotionID)
	if !exists {
		writeError(w, http.StatusNotFound, "promotion not found")
		return
	}

	if err := a.promotions.Delete(promotionID); err != nil {
		switch {
		case errors.Is(err, promotions.ErrInvalidPromotion):
			writeError(w, http.StatusBadRequest, "promotion id is required")
		default:
			writeError(w, http.StatusInternalServerError, "unable to delete promotion")
		}
		return
	}

	a.recordAuditLog(
		r,
		"promotion_deleted",
		"promotion",
		promotionID,
		previousPromotion,
		nil,
		nil,
	)
	w.WriteHeader(http.StatusNoContent)
}
