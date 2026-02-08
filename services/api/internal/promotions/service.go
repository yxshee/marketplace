package promotions

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/platform/identifier"
)

var (
	ErrPromotionNotFound  = errors.New("promotion not found")
	ErrInvalidPromotion   = errors.New("invalid promotion input")
	ErrNoPromotionChanges = errors.New("no promotion changes provided")
)

type Promotion struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	RuleJSON  json.RawMessage `json:"rule_json"`
	StartsAt  *time.Time      `json:"starts_at,omitempty"`
	EndsAt    *time.Time      `json:"ends_at,omitempty"`
	Stackable bool            `json:"stackable"`
	Active    bool            `json:"active"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type CreatePromotionInput struct {
	Name      string
	RuleJSON  json.RawMessage
	StartsAt  *time.Time
	EndsAt    *time.Time
	Stackable *bool
	Active    *bool
}

type UpdatePromotionInput struct {
	Name      *string
	RuleJSON  *json.RawMessage
	StartsAt  *time.Time
	EndsAt    *time.Time
	Stackable *bool
	Active    *bool
}

type Service struct {
	mu    sync.RWMutex
	byID  map[string]Promotion
	order []string
	now   func() time.Time
}

func NewService() *Service {
	return &Service{
		byID:  make(map[string]Promotion),
		order: make([]string, 0),
		now:   func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) List() []Promotion {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]Promotion, 0, len(s.order))
	for i := len(s.order) - 1; i >= 0; i-- {
		id := s.order[i]
		if promotion, exists := s.byID[id]; exists {
			items = append(items, promotion)
		}
	}

	return items
}

func (s *Service) Create(input CreatePromotionInput) (Promotion, error) {
	name, err := normalizePromotionName(input.Name)
	if err != nil {
		return Promotion{}, err
	}
	ruleJSON, err := normalizeRuleJSON(input.RuleJSON)
	if err != nil {
		return Promotion{}, err
	}
	if err := validateWindow(input.StartsAt, input.EndsAt); err != nil {
		return Promotion{}, err
	}

	active := true
	if input.Active != nil {
		active = *input.Active
	}
	stackable := false
	if input.Stackable != nil {
		stackable = *input.Stackable
	}

	now := s.now()
	promotion := Promotion{
		ID:        identifier.New("prm"),
		Name:      name,
		RuleJSON:  ruleJSON,
		StartsAt:  cloneTime(input.StartsAt),
		EndsAt:    cloneTime(input.EndsAt),
		Stackable: stackable,
		Active:    active,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[promotion.ID] = promotion
	s.order = append(s.order, promotion.ID)

	return promotion, nil
}

func (s *Service) Update(promotionID string, input UpdatePromotionInput) (Promotion, error) {
	id := strings.TrimSpace(promotionID)
	if id == "" {
		return Promotion{}, ErrInvalidPromotion
	}

	noChanges := input.Name == nil &&
		input.RuleJSON == nil &&
		input.StartsAt == nil &&
		input.EndsAt == nil &&
		input.Stackable == nil &&
		input.Active == nil
	if noChanges {
		return Promotion{}, ErrNoPromotionChanges
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	promotion, exists := s.byID[id]
	if !exists {
		return Promotion{}, ErrPromotionNotFound
	}

	if input.Name != nil {
		name, err := normalizePromotionName(*input.Name)
		if err != nil {
			return Promotion{}, err
		}
		promotion.Name = name
	}
	if input.RuleJSON != nil {
		ruleJSON, err := normalizeRuleJSON(*input.RuleJSON)
		if err != nil {
			return Promotion{}, err
		}
		promotion.RuleJSON = ruleJSON
	}
	if input.Stackable != nil {
		promotion.Stackable = *input.Stackable
	}
	if input.Active != nil {
		promotion.Active = *input.Active
	}

	startsAt := promotion.StartsAt
	endsAt := promotion.EndsAt
	if input.StartsAt != nil {
		startsAt = cloneTime(input.StartsAt)
	}
	if input.EndsAt != nil {
		endsAt = cloneTime(input.EndsAt)
	}
	if err := validateWindow(startsAt, endsAt); err != nil {
		return Promotion{}, err
	}
	promotion.StartsAt = startsAt
	promotion.EndsAt = endsAt

	promotion.UpdatedAt = s.now()
	s.byID[id] = promotion

	return promotion, nil
}

func (s *Service) Delete(promotionID string) error {
	id := strings.TrimSpace(promotionID)
	if id == "" {
		return ErrInvalidPromotion
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byID[id]; !exists {
		return ErrPromotionNotFound
	}
	delete(s.byID, id)

	filtered := s.order[:0]
	for _, orderedID := range s.order {
		if orderedID != id {
			filtered = append(filtered, orderedID)
		}
	}
	s.order = filtered

	return nil
}

func normalizePromotionName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if len(name) < 2 || len(name) > 120 {
		return "", ErrInvalidPromotion
	}
	return name, nil
}

func normalizeRuleJSON(raw json.RawMessage) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, ErrInvalidPromotion
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, ErrInvalidPromotion
	}
	if len(decoded) == 0 {
		return nil, ErrInvalidPromotion
	}

	canonical, err := json.Marshal(decoded)
	if err != nil {
		return nil, ErrInvalidPromotion
	}

	return canonical, nil
}

func validateWindow(startsAt, endsAt *time.Time) error {
	if startsAt == nil || endsAt == nil {
		return nil
	}
	if endsAt.Before(*startsAt) {
		return ErrInvalidPromotion
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
