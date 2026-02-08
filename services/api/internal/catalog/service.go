package catalog

import (
	"errors"
	"sync"
	"time"

	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/platform/identifier"
)

type ProductStatus string

const (
	ProductStatusDraft           ProductStatus = "draft"
	ProductStatusPendingApproval ProductStatus = "pending_approval"
	ProductStatusApproved        ProductStatus = "approved"
	ProductStatusRejected        ProductStatus = "rejected"
)

type ModerationDecision string

const (
	ModerationDecisionApprove ModerationDecision = "approve"
	ModerationDecisionReject  ModerationDecision = "reject"
)

var (
	ErrProductNotFound           = errors.New("product not found")
	ErrUnauthorizedProductAccess = errors.New("unauthorized product access")
	ErrInvalidStatusTransition   = errors.New("invalid status transition")
	ErrInvalidModerationDecision = errors.New("invalid moderation decision")
)

// Product models the core catalog aggregate used in the foundation branch.
type Product struct {
	ID                string
	VendorID          string
	OwnerUserID       string
	Title             string
	Description       string
	PriceInclTaxCents int64
	Currency          string
	StockQty          int32
	Status            ProductStatus
	ModerationReason  string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Service provides product and moderation workflow operations.
type Service struct {
	mu      sync.RWMutex
	byID    map[string]Product
	ordered []string
}

func NewService() *Service {
	return &Service{
		byID: make(map[string]Product),
	}
}

func (s *Service) CreateProduct(ownerUserID, vendorID, title, description, currency string, priceInclTaxCents int64) Product {
	now := time.Now().UTC()
	product := Product{
		ID:                identifier.New("prd"),
		OwnerUserID:       ownerUserID,
		VendorID:          vendorID,
		Title:             title,
		Description:       description,
		PriceInclTaxCents: priceInclTaxCents,
		Currency:          currency,
		StockQty:          0,
		Status:            ProductStatusDraft,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[product.ID] = product
	s.ordered = append(s.ordered, product.ID)
	return product
}

func (s *Service) SubmitForModeration(productID, ownerUserID, vendorID string) (Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	product, exists := s.byID[productID]
	if !exists {
		return Product{}, ErrProductNotFound
	}
	if product.OwnerUserID != ownerUserID || product.VendorID != vendorID {
		return Product{}, ErrUnauthorizedProductAccess
	}
	if product.Status != ProductStatusDraft && product.Status != ProductStatusRejected {
		return Product{}, ErrInvalidStatusTransition
	}

	product.Status = ProductStatusPendingApproval
	product.ModerationReason = ""
	product.UpdatedAt = time.Now().UTC()
	s.byID[productID] = product
	return product, nil
}

func (s *Service) ReviewProduct(productID, reviewerID string, decision ModerationDecision, reason string) (Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	product, exists := s.byID[productID]
	if !exists {
		return Product{}, ErrProductNotFound
	}
	if product.Status != ProductStatusPendingApproval {
		return Product{}, ErrInvalidStatusTransition
	}
	if reviewerID == "" {
		return Product{}, ErrUnauthorizedProductAccess
	}

	switch decision {
	case ModerationDecisionApprove:
		product.Status = ProductStatusApproved
		product.ModerationReason = ""
	case ModerationDecisionReject:
		product.Status = ProductStatusRejected
		product.ModerationReason = reason
	default:
		return Product{}, ErrInvalidModerationDecision
	}

	product.UpdatedAt = time.Now().UTC()
	s.byID[productID] = product
	return product, nil
}

func (s *Service) ListVisibleProducts(vendorVisible func(vendorID string) bool) []Product {
	s.mu.RLock()
	defer s.mu.RUnlock()

	visible := make([]Product, 0)
	for _, productID := range s.ordered {
		product := s.byID[productID]
		if product.Status != ProductStatusApproved {
			continue
		}
		if vendorVisible != nil && !vendorVisible(product.VendorID) {
			continue
		}
		visible = append(visible, product)
	}
	return visible
}
