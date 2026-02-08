package promotions

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPromotionCRUD(t *testing.T) {
	service := NewService()

	starts := time.Date(2026, 2, 8, 12, 0, 0, 0, time.UTC)
	ends := starts.Add(24 * time.Hour)

	created, err := service.Create(CreatePromotionInput{
		Name:      "Launch Discount",
		RuleJSON:  json.RawMessage(`{"type":"percentage","value":10}`),
		StartsAt:  &starts,
		EndsAt:    &ends,
		Stackable: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Name != "Launch Discount" {
		t.Fatalf("unexpected promotion name %s", created.Name)
	}
	if !created.Stackable || !created.Active {
		t.Fatalf("expected created promotion stackable=true active=true, got %#v", created)
	}

	list := service.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 promotion in list, got %d", len(list))
	}
	if list[0].ID != created.ID {
		t.Fatalf("expected created promotion id %s in list, got %s", created.ID, list[0].ID)
	}

	updated, err := service.Update(created.ID, UpdatePromotionInput{
		Name:      stringPtr("Launch Discount Final"),
		RuleJSON:  rawMessagePtr(json.RawMessage(`{"type":"percentage","value":15}`)),
		Stackable: boolPtr(false),
		Active:    boolPtr(false),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "Launch Discount Final" {
		t.Fatalf("expected updated name, got %s", updated.Name)
	}
	if updated.Stackable || updated.Active {
		t.Fatalf("expected updated stackable=false active=false, got %#v", updated)
	}

	if err := service.Delete(created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if len(service.List()) != 0 {
		t.Fatalf("expected no promotions after delete")
	}
}

func TestPromotionValidation(t *testing.T) {
	service := NewService()

	if _, err := service.Create(CreatePromotionInput{
		Name:     "X",
		RuleJSON: json.RawMessage(`{"type":"percentage","value":10}`),
	}); err != ErrInvalidPromotion {
		t.Fatalf("expected ErrInvalidPromotion for short name, got %v", err)
	}

	if _, err := service.Create(CreatePromotionInput{
		Name:     "No Rule",
		RuleJSON: json.RawMessage(`{}`),
	}); err != ErrInvalidPromotion {
		t.Fatalf("expected ErrInvalidPromotion for empty rule json, got %v", err)
	}

	starts := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	ends := starts.Add(-time.Hour)
	if _, err := service.Create(CreatePromotionInput{
		Name:     "Bad Window",
		RuleJSON: json.RawMessage(`{"type":"percentage","value":10}`),
		StartsAt: &starts,
		EndsAt:   &ends,
	}); err != ErrInvalidPromotion {
		t.Fatalf("expected ErrInvalidPromotion for invalid window, got %v", err)
	}

	created, err := service.Create(CreatePromotionInput{
		Name:     "Good Promotion",
		RuleJSON: json.RawMessage(`{"type":"fixed","value":500}`),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := service.Update(created.ID, UpdatePromotionInput{}); err != ErrNoPromotionChanges {
		t.Fatalf("expected ErrNoPromotionChanges, got %v", err)
	}

	if err := service.Delete("missing"); err != ErrPromotionNotFound {
		t.Fatalf("expected ErrPromotionNotFound on missing delete, got %v", err)
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func rawMessagePtr(value json.RawMessage) *json.RawMessage {
	return &value
}
