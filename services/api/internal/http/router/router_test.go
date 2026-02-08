package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/config"
)

func testConfig() config.Config {
	return config.Config{
		Port:             "8080",
		Environment:      "test",
		JWTSecret:        "test-secret",
		JWTIssuer:        "marketplace-api",
		AccessTokenTTL:   testSeconds(900),
		RefreshTokenTTL:  testSeconds(3600),
		SuperAdminEmails: "admin@example.com",
		SupportEmails:    "support@example.com",
		FinanceEmails:    "finance@example.com",
		CatalogModEmails: "moderator@example.com",
	}
}

func testSeconds(v int) time.Duration {
	return time.Duration(v) * time.Second
}

func mustRouter(t *testing.T) http.Handler {
	t.Helper()
	r, err := New(testConfig())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return r
}

func requestJSON(t *testing.T, r http.Handler, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewBuffer(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

type authPayload struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         struct {
		ID       string  `json:"id"`
		Role     string  `json:"role"`
		VendorID *string `json:"vendor_id"`
	} `json:"user"`
}

func registerUser(t *testing.T, r http.Handler, email string) authPayload {
	t.Helper()
	rr := requestJSON(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": "strong-password",
	}, "")
	if rr.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload authPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return payload
}

func loginUser(t *testing.T, r http.Handler, email string) authPayload {
	t.Helper()
	rr := requestJSON(t, r, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": "strong-password",
	}, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload authPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return payload
}

func TestAuthRegisterLoginMeAndRefreshRotation(t *testing.T) {
	r := mustRouter(t)
	registered := registerUser(t, r, "buyer@example.com")

	me := requestJSON(t, r, http.MethodGet, "/api/v1/auth/me", nil, registered.AccessToken)
	if me.Code != http.StatusOK {
		t.Fatalf("auth me status=%d body=%s", me.Code, me.Body.String())
	}

	refresh := requestJSON(t, r, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": registered.RefreshToken,
	}, "")
	if refresh.Code != http.StatusOK {
		t.Fatalf("refresh status=%d body=%s", refresh.Code, refresh.Body.String())
	}

	var refreshed authPayload
	if err := json.Unmarshal(refresh.Body.Bytes(), &refreshed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if refreshed.RefreshToken == registered.RefreshToken {
		t.Fatal("expected refresh token rotation")
	}

	secondRefresh := requestJSON(t, r, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": registered.RefreshToken,
	}, "")
	if secondRefresh.Code != http.StatusUnauthorized {
		t.Fatalf("expected old refresh token to fail, got status=%d body=%s", secondRefresh.Code, secondRefresh.Body.String())
	}
}

func TestRBACSupportAndFinanceSegmentation(t *testing.T) {
	r := mustRouter(t)

	support := registerUser(t, r, "support@example.com")
	finance := registerUser(t, r, "finance@example.com")

	supportCommission := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/vendors/ven_missing/commission", map[string]int32{
		"commission_override_bps": 900,
	}, support.AccessToken)
	if supportCommission.Code != http.StatusForbidden {
		t.Fatalf("expected support to be forbidden for commission, got status=%d body=%s", supportCommission.Code, supportCommission.Body.String())
	}

	financeModeration := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/moderation/products/prd_missing", map[string]string{
		"decision": "approve",
	}, finance.AccessToken)
	if financeModeration.Code != http.StatusForbidden {
		t.Fatalf("expected finance to be forbidden for moderation, got status=%d body=%s", financeModeration.Code, financeModeration.Body.String())
	}
}

func TestModerationWorkflowSkeleton(t *testing.T) {
	r := mustRouter(t)

	buyer := registerUser(t, r, "vendor-owner@example.com")
	admin := registerUser(t, r, "admin@example.com")
	moderator := registerUser(t, r, "moderator@example.com")

	vendorCreated := requestJSON(t, r, http.MethodPost, "/api/v1/vendors/register", map[string]string{
		"slug":         "vendor-one",
		"display_name": "Vendor One",
	}, buyer.AccessToken)
	if vendorCreated.Code != http.StatusCreated {
		t.Fatalf("vendor register status=%d body=%s", vendorCreated.Code, vendorCreated.Body.String())
	}

	var vendorBody struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(vendorCreated.Body.Bytes(), &vendorBody); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	verified := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/vendors/"+vendorBody.ID+"/verification", map[string]string{
		"state":  "verified",
		"reason": "kyc complete",
	}, admin.AccessToken)
	if verified.Code != http.StatusOK {
		t.Fatalf("admin verify vendor status=%d body=%s", verified.Code, verified.Body.String())
	}

	vendorLogin := loginUser(t, r, "vendor-owner@example.com")
	createdProduct := requestJSON(t, r, http.MethodPost, "/api/v1/vendor/products", map[string]interface{}{
		"title":                "Notebook",
		"description":          "Thin notebook",
		"price_incl_tax_cents": 2499,
		"currency":             "USD",
	}, vendorLogin.AccessToken)
	if createdProduct.Code != http.StatusCreated {
		t.Fatalf("create product status=%d body=%s", createdProduct.Code, createdProduct.Body.String())
	}

	var product struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createdProduct.Body.Bytes(), &product); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	submitted := requestJSON(t, r, http.MethodPost, "/api/v1/vendor/products/"+product.ID+"/submit-moderation", map[string]string{}, vendorLogin.AccessToken)
	if submitted.Code != http.StatusOK {
		t.Fatalf("submit moderation status=%d body=%s", submitted.Code, submitted.Body.String())
	}

	approved := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/moderation/products/"+product.ID, map[string]string{
		"decision": "approve",
	}, moderator.AccessToken)
	if approved.Code != http.StatusOK {
		t.Fatalf("approve moderation status=%d body=%s", approved.Code, approved.Body.String())
	}

	catalog := requestJSON(t, r, http.MethodGet, "/api/v1/catalog/products", nil, "")
	if catalog.Code != http.StatusOK {
		t.Fatalf("catalog status=%d body=%s", catalog.Code, catalog.Body.String())
	}

	var catalogBody struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(catalog.Body.Bytes(), &catalogBody); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if catalogBody.Total != 1 {
		t.Fatalf("expected 1 catalog item, got %d", catalogBody.Total)
	}
}
