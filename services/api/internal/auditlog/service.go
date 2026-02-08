package auditlog

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/platform/identifier"
)

var ErrInvalidAuditLog = errors.New("invalid audit log input")

type Entry struct {
	ID           string          `json:"id"`
	ActorType    string          `json:"actor_type"`
	ActorID      string          `json:"actor_id"`
	ActorRole    string          `json:"actor_role,omitempty"`
	Action       string          `json:"action"`
	TargetType   string          `json:"target_type"`
	TargetID     string          `json:"target_id"`
	BeforeJSON   json.RawMessage `json:"before_json,omitempty"`
	AfterJSON    json.RawMessage `json:"after_json,omitempty"`
	MetadataJSON json.RawMessage `json:"metadata_json,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

type RecordInput struct {
	ActorType  string
	ActorID    string
	ActorRole  string
	Action     string
	TargetType string
	TargetID   string
	Before     interface{}
	After      interface{}
	Metadata   interface{}
}

type ListInput struct {
	ActorType  string
	ActorID    string
	Action     string
	TargetType string
	TargetID   string
	Limit      int
	Offset     int
}

type ListResult struct {
	Items []Entry `json:"items"`
	Total int     `json:"total"`
}

type Service struct {
	mu    sync.RWMutex
	byID  map[string]Entry
	order []string
	now   func() time.Time
}

func NewService() *Service {
	return &Service{
		byID:  make(map[string]Entry),
		order: make([]string, 0),
		now:   func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Record(input RecordInput) (Entry, error) {
	actorType, err := normalizeRequired(input.ActorType, true)
	if err != nil {
		return Entry{}, err
	}
	actorID, err := normalizeRequired(input.ActorID, false)
	if err != nil {
		return Entry{}, err
	}
	action, err := normalizeRequired(input.Action, true)
	if err != nil {
		return Entry{}, err
	}
	targetType, err := normalizeRequired(input.TargetType, true)
	if err != nil {
		return Entry{}, err
	}
	targetID, err := normalizeRequired(input.TargetID, false)
	if err != nil {
		return Entry{}, err
	}

	beforeJSON, err := normalizeOptionalJSON(input.Before)
	if err != nil {
		return Entry{}, err
	}
	afterJSON, err := normalizeOptionalJSON(input.After)
	if err != nil {
		return Entry{}, err
	}
	metadataJSON, err := normalizeOptionalJSON(input.Metadata)
	if err != nil {
		return Entry{}, err
	}

	entry := Entry{
		ID:           identifier.New("aud"),
		ActorType:    actorType,
		ActorID:      actorID,
		ActorRole:    strings.ToLower(strings.TrimSpace(input.ActorRole)),
		Action:       action,
		TargetType:   targetType,
		TargetID:     targetID,
		BeforeJSON:   beforeJSON,
		AfterJSON:    afterJSON,
		MetadataJSON: metadataJSON,
		CreatedAt:    s.now(),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[entry.ID] = entry
	s.order = append(s.order, entry.ID)
	return entry, nil
}

func (s *Service) List(input ListInput) ListResult {
	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	actorType := strings.ToLower(strings.TrimSpace(input.ActorType))
	actorID := strings.TrimSpace(input.ActorID)
	action := strings.ToLower(strings.TrimSpace(input.Action))
	targetType := strings.ToLower(strings.TrimSpace(input.TargetType))
	targetID := strings.TrimSpace(input.TargetID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	matches := make([]Entry, 0, len(s.order))
	for i := len(s.order) - 1; i >= 0; i-- {
		id := s.order[i]
		entry, exists := s.byID[id]
		if !exists {
			continue
		}
		if actorType != "" && entry.ActorType != actorType {
			continue
		}
		if actorID != "" && entry.ActorID != actorID {
			continue
		}
		if action != "" && entry.Action != action {
			continue
		}
		if targetType != "" && entry.TargetType != targetType {
			continue
		}
		if targetID != "" && entry.TargetID != targetID {
			continue
		}
		matches = append(matches, entry)
	}

	total := len(matches)
	if offset >= total {
		return ListResult{Items: []Entry{}, Total: total}
	}

	end := offset + limit
	if end > total {
		end = total
	}

	items := make([]Entry, end-offset)
	copy(items, matches[offset:end])
	return ListResult{Items: items, Total: total}
}

func normalizeRequired(raw string, forceLower bool) (string, error) {
	value := strings.TrimSpace(raw)
	if forceLower {
		value = strings.ToLower(value)
	}
	if value == "" {
		return "", ErrInvalidAuditLog
	}
	return value, nil
}

func normalizeOptionalJSON(value interface{}) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}

	switch typed := value.(type) {
	case json.RawMessage:
		if len(strings.TrimSpace(string(typed))) == 0 {
			return nil, nil
		}
		var decoded interface{}
		if err := json.Unmarshal(typed, &decoded); err != nil {
			return nil, ErrInvalidAuditLog
		}
		encoded, err := json.Marshal(decoded)
		if err != nil {
			return nil, ErrInvalidAuditLog
		}
		return json.RawMessage(encoded), nil
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, ErrInvalidAuditLog
		}
		return json.RawMessage(encoded), nil
	}
}
