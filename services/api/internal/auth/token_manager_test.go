package auth

import (
	"testing"
	"time"
)

func TestIssueAndParseTokenPair(t *testing.T) {
	manager, err := NewTokenManager("test-secret", "marketplace-api", testDuration(900), testDuration(3600))
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	user := User{ID: "usr_1", Role: RoleBuyer}
	pair, err := manager.IssueTokenPair(user, "ses_1")
	if err != nil {
		t.Fatalf("IssueTokenPair() error = %v", err)
	}

	accessClaims, err := manager.ParseAndValidate(pair.AccessToken, TokenTypeAccess)
	if err != nil {
		t.Fatalf("ParseAndValidate(access) error = %v", err)
	}
	if accessClaims.UserID != user.ID || accessClaims.Role != user.Role || accessClaims.SessionID != "ses_1" {
		t.Fatalf("unexpected claims: %+v", accessClaims)
	}

	if _, err := manager.ParseAndValidate(pair.AccessToken, TokenTypeRefresh); err != ErrInvalidTokenType {
		t.Fatalf("expected ErrInvalidTokenType, got %v", err)
	}

	refreshClaims, err := manager.ParseAndValidate(pair.RefreshToken, TokenTypeRefresh)
	if err != nil {
		t.Fatalf("ParseAndValidate(refresh) error = %v", err)
	}
	if refreshClaims.SessionID != "ses_1" {
		t.Fatalf("unexpected session id: %s", refreshClaims.SessionID)
	}
}

func TestParseInvalidSignature(t *testing.T) {
	good, err := NewTokenManager("good-secret", "marketplace-api", testDuration(900), testDuration(3600))
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	bad, err := NewTokenManager("bad-secret", "marketplace-api", testDuration(900), testDuration(3600))
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	pair, err := good.IssueTokenPair(User{ID: "usr_1", Role: RoleBuyer}, "ses_1")
	if err != nil {
		t.Fatalf("IssueTokenPair() error = %v", err)
	}

	if _, err := bad.ParseAndValidate(pair.AccessToken, TokenTypeAccess); err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func testDuration(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}
