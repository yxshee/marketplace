package auditlog

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRecordAndList(t *testing.T) {
	svc := NewService()
	clock := []time.Time{
		time.Date(2026, time.January, 1, 10, 0, 0, 0, time.UTC),
		time.Date(2026, time.January, 1, 10, 1, 0, 0, time.UTC),
	}
	idx := 0
	svc.now = func() time.Time {
		value := clock[idx]
		if idx < len(clock)-1 {
			idx++
		}
		return value
	}

	first, err := svc.Record(RecordInput{
		ActorType:  "admin",
		ActorID:    "usr_admin_one",
		ActorRole:  "support",
		Action:     "vendor_verification_updated",
		TargetType: "vendor",
		TargetID:   "ven_001",
		Before: map[string]string{
			"verification_state": "pending",
		},
		After: map[string]string{
			"verification_state": "verified",
		},
		Metadata: map[string]string{
			"reason": "kyc complete",
		},
	})
	if err != nil {
		t.Fatalf("Record() first error = %v", err)
	}
	if first.ID == "" {
		t.Fatalf("expected first record id")
	}

	second, err := svc.Record(RecordInput{
		ActorType:  "admin",
		ActorID:    "usr_finance",
		ActorRole:  "finance",
		Action:     "promotion_created",
		TargetType: "promotion",
		TargetID:   "prm_001",
		After: map[string]interface{}{
			"name": "Spring Sale",
		},
	})
	if err != nil {
		t.Fatalf("Record() second error = %v", err)
	}
	if second.ID == "" {
		t.Fatalf("expected second record id")
	}

	all := svc.List(ListInput{})
	if all.Total != 2 || len(all.Items) != 2 {
		t.Fatalf("expected two audit logs, got total=%d len=%d", all.Total, len(all.Items))
	}
	if all.Items[0].ID != second.ID {
		t.Fatalf("expected newest item first, got %s", all.Items[0].ID)
	}
	if all.Items[1].ID != first.ID {
		t.Fatalf("expected oldest item second, got %s", all.Items[1].ID)
	}

	financeOnly := svc.List(ListInput{ActorID: "usr_finance"})
	if financeOnly.Total != 1 || len(financeOnly.Items) != 1 {
		t.Fatalf("expected one finance item, got total=%d len=%d", financeOnly.Total, len(financeOnly.Items))
	}
	if financeOnly.Items[0].ID != second.ID {
		t.Fatalf("expected finance filter to return second item, got %s", financeOnly.Items[0].ID)
	}

	actionFilter := svc.List(ListInput{Action: "PROMOTION_CREATED"})
	if actionFilter.Total != 1 || len(actionFilter.Items) != 1 {
		t.Fatalf("expected one promotion_created action, got total=%d len=%d", actionFilter.Total, len(actionFilter.Items))
	}
	if actionFilter.Items[0].ID != second.ID {
		t.Fatalf("expected action filter to return second item, got %s", actionFilter.Items[0].ID)
	}

	paginated := svc.List(ListInput{Limit: 1, Offset: 1})
	if paginated.Total != 2 || len(paginated.Items) != 1 {
		t.Fatalf("expected paginated result total=2 len=1, got total=%d len=%d", paginated.Total, len(paginated.Items))
	}
	if paginated.Items[0].ID != first.ID {
		t.Fatalf("expected pagination to return first item on offset 1, got %s", paginated.Items[0].ID)
	}

	var afterPayload map[string]interface{}
	if err := json.Unmarshal(first.AfterJSON, &afterPayload); err != nil {
		t.Fatalf("json.Unmarshal(after) error = %v", err)
	}
	if afterPayload["verification_state"] != "verified" {
		t.Fatalf("expected after payload verification_state=verified, got %#v", afterPayload)
	}
}

func TestRecordValidation(t *testing.T) {
	svc := NewService()

	_, err := svc.Record(RecordInput{
		ActorType:  "",
		ActorID:    "usr_1",
		Action:     "x",
		TargetType: "y",
		TargetID:   "z",
	})
	if err == nil {
		t.Fatalf("expected missing actor type to fail")
	}

	_, err = svc.Record(RecordInput{
		ActorType:  "admin",
		ActorID:    "usr_1",
		Action:     "x",
		TargetType: "y",
		TargetID:   "z",
		Before:     json.RawMessage("{"),
	})
	if err == nil {
		t.Fatalf("expected invalid raw json payload to fail")
	}
}
