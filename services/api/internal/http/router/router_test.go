package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
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

func mustRouterWithConfig(t *testing.T, cfg config.Config) http.Handler {
	t.Helper()
	r, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return r
}

func requestJSON(t *testing.T, r http.Handler, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	return requestJSONWithHeaders(t, r, method, path, body, token, nil)
}

func requestJSONWithHeaders(
	t *testing.T,
	r http.Handler,
	method, path string,
	body interface{},
	token string,
	headers map[string]string,
) *httptest.ResponseRecorder {
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
	for key, value := range headers {
		req.Header.Set(key, value)
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

func TestCheckoutCreatesMultiShipmentAndIdempotentOrder(t *testing.T) {
	cfg := testConfig()
	cfg.Environment = "development"
	r := mustRouterWithConfig(t, cfg)

	catalogRes := requestJSON(t, r, http.MethodGet, "/api/v1/catalog/products", nil, "")
	if catalogRes.Code != http.StatusOK {
		t.Fatalf("catalog status=%d body=%s", catalogRes.Code, catalogRes.Body.String())
	}

	var catalogPayload struct {
		Items []struct {
			ID       string `json:"id"`
			VendorID string `json:"vendor_id"`
			Price    int64  `json:"price_incl_tax_cents"`
		} `json:"items"`
	}
	if err := json.Unmarshal(catalogRes.Body.Bytes(), &catalogPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(catalogPayload.Items) < 2 {
		t.Fatalf("expected seeded catalog items, got %d", len(catalogPayload.Items))
	}

	productByVendor := make(map[string]struct {
		ID    string
		Price int64
	})
	for _, item := range catalogPayload.Items {
		if _, exists := productByVendor[item.VendorID]; !exists {
			productByVendor[item.VendorID] = struct {
				ID    string
				Price int64
			}{ID: item.ID, Price: item.Price}
		}
	}
	if len(productByVendor) < 2 {
		t.Fatalf("expected at least two vendors in seeded catalog, got %d", len(productByVendor))
	}

	vendorIDs := make([]string, 0, len(productByVendor))
	for vendorID := range productByVendor {
		vendorIDs = append(vendorIDs, vendorID)
	}
	sort.Strings(vendorIDs)

	guestHeaders := map[string]string{guestTokenHeader: "gst_test_checkout_flow"}
	selectedProducts := []struct {
		ID    string
		Price int64
	}{
		productByVendor[vendorIDs[0]],
		productByVendor[vendorIDs[1]],
	}

	var expectedSubtotal int64
	for _, selected := range selectedProducts {
		addRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/cart/items", map[string]interface{}{
			"product_id": selected.ID,
			"qty":        1,
		}, "", guestHeaders)
		if addRes.Code != http.StatusOK {
			t.Fatalf("add cart item status=%d body=%s", addRes.Code, addRes.Body.String())
		}
		expectedSubtotal += selected.Price
	}

	quoteRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/checkout/quote", map[string]interface{}{}, "", guestHeaders)
	if quoteRes.Code != http.StatusOK {
		t.Fatalf("quote status=%d body=%s", quoteRes.Code, quoteRes.Body.String())
	}

	var quotePayload struct {
		ShipmentCount int32 `json:"shipment_count"`
		SubtotalCents int64 `json:"subtotal_cents"`
		ShippingCents int64 `json:"shipping_cents"`
		TotalCents    int64 `json:"total_cents"`
	}
	if err := json.Unmarshal(quoteRes.Body.Bytes(), &quotePayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if quotePayload.ShipmentCount != 2 {
		t.Fatalf("expected shipment_count=2, got %d", quotePayload.ShipmentCount)
	}
	if quotePayload.SubtotalCents != expectedSubtotal {
		t.Fatalf("expected subtotal=%d, got %d", expectedSubtotal, quotePayload.SubtotalCents)
	}
	if quotePayload.ShippingCents != 1000 {
		t.Fatalf("expected shipping=1000, got %d", quotePayload.ShippingCents)
	}
	if quotePayload.TotalCents != expectedSubtotal+1000 {
		t.Fatalf("expected total=%d, got %d", expectedSubtotal+1000, quotePayload.TotalCents)
	}

	placeRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/checkout/place-order", map[string]interface{}{
		"idempotency_key": "idem-test-order-1",
	}, "", guestHeaders)
	if placeRes.Code != http.StatusCreated {
		t.Fatalf("place order status=%d body=%s", placeRes.Code, placeRes.Body.String())
	}

	var placePayload struct {
		Order struct {
			ID            string `json:"id"`
			ShipmentCount int32  `json:"shipment_count"`
			TotalCents    int64  `json:"total_cents"`
		} `json:"order"`
	}
	if err := json.Unmarshal(placeRes.Body.Bytes(), &placePayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if placePayload.Order.ShipmentCount != 2 {
		t.Fatalf("expected order shipment_count=2, got %d", placePayload.Order.ShipmentCount)
	}
	if placePayload.Order.TotalCents != expectedSubtotal+1000 {
		t.Fatalf("expected order total=%d, got %d", expectedSubtotal+1000, placePayload.Order.TotalCents)
	}

	retryRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/checkout/place-order", map[string]interface{}{
		"idempotency_key": "idem-test-order-1",
	}, "", guestHeaders)
	if retryRes.Code != http.StatusCreated {
		t.Fatalf("retry place order status=%d body=%s", retryRes.Code, retryRes.Body.String())
	}

	var retryPayload struct {
		Order struct {
			ID string `json:"id"`
		} `json:"order"`
	}
	if err := json.Unmarshal(retryRes.Body.Bytes(), &retryPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if retryPayload.Order.ID != placePayload.Order.ID {
		t.Fatalf("expected idempotent order id %s, got %s", placePayload.Order.ID, retryPayload.Order.ID)
	}

	cartRes := requestJSONWithHeaders(t, r, http.MethodGet, "/api/v1/cart", nil, "", guestHeaders)
	if cartRes.Code != http.StatusOK {
		t.Fatalf("cart status=%d body=%s", cartRes.Code, cartRes.Body.String())
	}

	var cartPayload struct {
		ItemCount int32 `json:"item_count"`
	}
	if err := json.Unmarshal(cartRes.Body.Bytes(), &cartPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if cartPayload.ItemCount != 0 {
		t.Fatalf("expected empty cart after order placement, got item_count=%d", cartPayload.ItemCount)
	}
}
