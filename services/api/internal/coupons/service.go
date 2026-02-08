package coupons

import (
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/yxshee/marketplace-platform/services/api/internal/platform/identifier"
)

type DiscountType string

const (
	DiscountTypePercent     DiscountType = "percent"
	DiscountTypeAmountCents DiscountType = "amount_cents"
)

var (
	ErrCouponNotFound          = errors.New("coupon not found")
	ErrCouponCodeInUse         = errors.New("coupon code already in use")
	ErrUnauthorizedCouponScope = errors.New("unauthorized coupon access")
	ErrInvalidCouponInput      = errors.New("invalid coupon input")
)

var couponCodePattern = regexp.MustCompile(`^[A-Z0-9_-]{3,32}$`)

type Coupon struct {
	ID            string       `json:"id"`
	VendorID      string       `json:"vendor_id"`
	Code          string       `json:"code"`
	DiscountType  DiscountType `json:"discount_type"`
	DiscountValue int64        `json:"discount_value"`
	StartsAt      *time.Time   `json:"starts_at,omitempty"`
	EndsAt        *time.Time   `json:"ends_at,omitempty"`
	UsageLimit    *int32       `json:"usage_limit,omitempty"`
	Active        bool         `json:"active"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

type CreateCouponInput struct {
	Code          string
	DiscountType  DiscountType
	DiscountValue int64
	StartsAt      *time.Time
	EndsAt        *time.Time
	UsageLimit    *int32
	Active        *bool
}

type UpdateCouponInput struct {
	Code          *string
	DiscountType  *DiscountType
	DiscountValue *int64
	Active        *bool
}

type Service struct {
	mu              sync.RWMutex
	byID            map[string]Coupon
	vendorOrder     map[string][]string
	vendorCodeIndex map[string]map[string]string
	now             func() time.Time
}

func NewService() *Service {
	return &Service{
		byID:            make(map[string]Coupon),
		vendorOrder:     make(map[string][]string),
		vendorCodeIndex: make(map[string]map[string]string),
		now:             func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) ListByVendor(vendorID string) []Coupon {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.vendorOrder[vendorID]
	items := make([]Coupon, 0, len(ids))
	for i := len(ids) - 1; i >= 0; i-- {
		id := ids[i]
		if coupon, exists := s.byID[id]; exists {
			items = append(items, coupon)
		}
	}
	return items
}

func (s *Service) Create(vendorID string, input CreateCouponInput) (Coupon, error) {
	if strings.TrimSpace(vendorID) == "" {
		return Coupon{}, ErrInvalidCouponInput
	}

	code, err := normalizeCode(input.Code)
	if err != nil {
		return Coupon{}, err
	}
	if err := validateDiscount(input.DiscountType, input.DiscountValue); err != nil {
		return Coupon{}, err
	}
	if err := validateWindow(input.StartsAt, input.EndsAt); err != nil {
		return Coupon{}, err
	}
	if input.UsageLimit != nil && *input.UsageLimit <= 0 {
		return Coupon{}, ErrInvalidCouponInput
	}

	active := true
	if input.Active != nil {
		active = *input.Active
	}

	now := s.now()
	coupon := Coupon{
		ID:            identifier.New("cpn"),
		VendorID:      vendorID,
		Code:          code,
		DiscountType:  input.DiscountType,
		DiscountValue: input.DiscountValue,
		StartsAt:      cloneTime(input.StartsAt),
		EndsAt:        cloneTime(input.EndsAt),
		UsageLimit:    cloneInt32(input.UsageLimit),
		Active:        active,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byCode(vendorID, code); exists {
		return Coupon{}, ErrCouponCodeInUse
	}

	s.byID[coupon.ID] = coupon
	s.vendorOrder[vendorID] = append(s.vendorOrder[vendorID], coupon.ID)
	s.ensureCodeIndex(vendorID)[code] = coupon.ID
	return coupon, nil
}

func (s *Service) Update(vendorID, couponID string, input UpdateCouponInput) (Coupon, error) {
	if strings.TrimSpace(vendorID) == "" || strings.TrimSpace(couponID) == "" {
		return Coupon{}, ErrInvalidCouponInput
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	coupon, exists := s.byID[couponID]
	if !exists {
		return Coupon{}, ErrCouponNotFound
	}
	if coupon.VendorID != vendorID {
		return Coupon{}, ErrUnauthorizedCouponScope
	}

	updatedCode := coupon.Code
	if input.Code != nil {
		normalized, err := normalizeCode(*input.Code)
		if err != nil {
			return Coupon{}, err
		}
		if existingID, exists := s.byCode(vendorID, normalized); exists && existingID != coupon.ID {
			return Coupon{}, ErrCouponCodeInUse
		}
		updatedCode = normalized
	}

	updatedType := coupon.DiscountType
	if input.DiscountType != nil {
		updatedType = *input.DiscountType
	}

	updatedValue := coupon.DiscountValue
	if input.DiscountValue != nil {
		updatedValue = *input.DiscountValue
	}
	if err := validateDiscount(updatedType, updatedValue); err != nil {
		return Coupon{}, err
	}

	if input.Active != nil {
		coupon.Active = *input.Active
	}
	coupon.Code = updatedCode
	coupon.DiscountType = updatedType
	coupon.DiscountValue = updatedValue
	coupon.UpdatedAt = s.now()
	s.byID[coupon.ID] = coupon

	codeIndex := s.ensureCodeIndex(vendorID)
	for code, indexedID := range codeIndex {
		if indexedID == coupon.ID {
			delete(codeIndex, code)
		}
	}
	codeIndex[coupon.Code] = coupon.ID

	return coupon, nil
}

func (s *Service) Delete(vendorID, couponID string) error {
	if strings.TrimSpace(vendorID) == "" || strings.TrimSpace(couponID) == "" {
		return ErrInvalidCouponInput
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	coupon, exists := s.byID[couponID]
	if !exists {
		return ErrCouponNotFound
	}
	if coupon.VendorID != vendorID {
		return ErrUnauthorizedCouponScope
	}

	delete(s.byID, couponID)
	if codeIndex, exists := s.vendorCodeIndex[vendorID]; exists {
		delete(codeIndex, coupon.Code)
	}

	ordered := s.vendorOrder[vendorID]
	filtered := ordered[:0]
	for _, id := range ordered {
		if id != couponID {
			filtered = append(filtered, id)
		}
	}
	s.vendorOrder[vendorID] = filtered

	return nil
}

func (s *Service) byCode(vendorID, code string) (string, bool) {
	index, exists := s.vendorCodeIndex[vendorID]
	if !exists {
		return "", false
	}
	value, exists := index[code]
	return value, exists
}

func (s *Service) ensureCodeIndex(vendorID string) map[string]string {
	index, exists := s.vendorCodeIndex[vendorID]
	if !exists {
		index = make(map[string]string)
		s.vendorCodeIndex[vendorID] = index
	}
	return index
}

func normalizeCode(raw string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if !couponCodePattern.MatchString(code) {
		return "", ErrInvalidCouponInput
	}
	return code, nil
}

func validateDiscount(discountType DiscountType, value int64) error {
	switch discountType {
	case DiscountTypePercent:
		if value <= 0 || value > 100 {
			return ErrInvalidCouponInput
		}
	case DiscountTypeAmountCents:
		if value <= 0 {
			return ErrInvalidCouponInput
		}
	default:
		return ErrInvalidCouponInput
	}
	return nil
}

func validateWindow(startsAt, endsAt *time.Time) error {
	if startsAt == nil || endsAt == nil {
		return nil
	}
	if endsAt.Before(*startsAt) {
		return ErrInvalidCouponInput
	}
	return nil
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func cloneInt32(value *int32) *int32 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
