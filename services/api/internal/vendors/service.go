package vendors

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yxshee/marketplace-platform/services/api/internal/platform/identifier"
)

type VerificationState string

const (
	VerificationPending   VerificationState = "pending"
	VerificationVerified  VerificationState = "verified"
	VerificationRejected  VerificationState = "rejected"
	VerificationSuspended VerificationState = "suspended"
)

var (
	ErrOwnerAlreadyVendor = errors.New("owner already has vendor")
	ErrSlugInUse          = errors.New("vendor slug already in use")
	ErrVendorNotFound     = errors.New("vendor not found")
	ErrInvalidState       = errors.New("invalid verification state")
)

// Vendor captures vendor profile and verification state.
type Vendor struct {
	ID                    string            `json:"id"`
	OwnerUserID           string            `json:"owner_user_id"`
	Slug                  string            `json:"slug"`
	DisplayName           string            `json:"display_name"`
	VerificationState     VerificationState `json:"verification_state"`
	CommissionOverrideBPS *int32            `json:"commission_override_bps"`
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
}

// Service provides in-memory vendor operations for the API foundation phase.
type Service struct {
	mu        sync.RWMutex
	byID      map[string]Vendor
	byOwnerID map[string]string
	bySlug    map[string]string
}

func NewService() *Service {
	return &Service{
		byID:      make(map[string]Vendor),
		byOwnerID: make(map[string]string),
		bySlug:    make(map[string]string),
	}
}

func (s *Service) Register(ownerUserID, slug, displayName string) (Vendor, error) {
	normalizedSlug := strings.ToLower(strings.TrimSpace(slug))
	if normalizedSlug == "" {
		return Vendor{}, ErrSlugInUse
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byOwnerID[ownerUserID]; exists {
		return Vendor{}, ErrOwnerAlreadyVendor
	}
	if _, exists := s.bySlug[normalizedSlug]; exists {
		return Vendor{}, ErrSlugInUse
	}

	now := time.Now().UTC()
	vendor := Vendor{
		ID:                identifier.New("ven"),
		OwnerUserID:       ownerUserID,
		Slug:              normalizedSlug,
		DisplayName:       strings.TrimSpace(displayName),
		VerificationState: VerificationPending,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	s.byID[vendor.ID] = vendor
	s.byOwnerID[ownerUserID] = vendor.ID
	s.bySlug[normalizedSlug] = vendor.ID
	return vendor, nil
}

func (s *Service) GetByOwner(ownerUserID string) (Vendor, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vendorID, exists := s.byOwnerID[ownerUserID]
	if !exists {
		return Vendor{}, false
	}
	vendor := s.byID[vendorID]
	return vendor, true
}

func (s *Service) GetByID(vendorID string) (Vendor, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vendor, exists := s.byID[vendorID]
	return vendor, exists
}

func (s *Service) List(verificationStateFilter *VerificationState) []Vendor {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]Vendor, 0, len(s.byID))
	for _, vendor := range s.byID {
		if verificationStateFilter != nil && vendor.VerificationState != *verificationStateFilter {
			continue
		}
		items = append(items, vendor)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})

	return items
}

func (s *Service) SetVerificationState(vendorID string, state VerificationState) (Vendor, error) {
	if !isValidState(state) {
		return Vendor{}, ErrInvalidState
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	vendor, exists := s.byID[vendorID]
	if !exists {
		return Vendor{}, ErrVendorNotFound
	}

	vendor.VerificationState = state
	vendor.UpdatedAt = time.Now().UTC()
	s.byID[vendorID] = vendor
	return vendor, nil
}

func (s *Service) SetCommission(vendorID string, commissionBPS int32) (Vendor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	vendor, exists := s.byID[vendorID]
	if !exists {
		return Vendor{}, ErrVendorNotFound
	}

	vendor.CommissionOverrideBPS = &commissionBPS
	vendor.UpdatedAt = time.Now().UTC()
	s.byID[vendorID] = vendor
	return vendor, nil
}

func isValidState(state VerificationState) bool {
	switch state {
	case VerificationPending, VerificationVerified, VerificationRejected, VerificationSuspended:
		return true
	default:
		return false
	}
}
