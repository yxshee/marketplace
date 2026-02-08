package vendors

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/platform/identifier"
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
	ID                    string
	OwnerUserID           string
	Slug                  string
	DisplayName           string
	VerificationState     VerificationState
	CommissionOverrideBPS *int32
	CreatedAt             time.Time
	UpdatedAt             time.Time
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
