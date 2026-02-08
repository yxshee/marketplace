package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v83/webhook"
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

func TestVendorOnboardingSelfServeLifecycle(t *testing.T) {
	r := mustRouter(t)
	owner := registerUser(t, r, "vendor-self@example.com")

	before := requestJSON(t, r, http.MethodGet, "/api/v1/vendor/verification-status", nil, owner.AccessToken)
	if before.Code != http.StatusNotFound {
		t.Fatalf("expected verification status before registration to be 404, got status=%d body=%s", before.Code, before.Body.String())
	}

	created := requestJSON(t, r, http.MethodPost, "/api/v1/vendors/register", map[string]string{
		"slug":         "north-studio",
		"display_name": "North Studio",
	}, owner.AccessToken)
	if created.Code != http.StatusCreated {
		t.Fatalf("vendor register status=%d body=%s", created.Code, created.Body.String())
	}

	var createdPayload struct {
		ID                string `json:"id"`
		Slug              string `json:"slug"`
		DisplayName       string `json:"display_name"`
		VerificationState string `json:"verification_state"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &createdPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if createdPayload.VerificationState != "pending" {
		t.Fatalf("expected pending verification state, got %s", createdPayload.VerificationState)
	}

	profile := requestJSON(t, r, http.MethodGet, "/api/v1/vendor/profile", nil, owner.AccessToken)
	if profile.Code != http.StatusOK {
		t.Fatalf("vendor profile status=%d body=%s", profile.Code, profile.Body.String())
	}

	status := requestJSON(t, r, http.MethodGet, "/api/v1/vendor/verification-status", nil, owner.AccessToken)
	if status.Code != http.StatusOK {
		t.Fatalf("verification status after registration status=%d body=%s", status.Code, status.Body.String())
	}

	var statusPayload struct {
		ID                string `json:"id"`
		Slug              string `json:"slug"`
		DisplayName       string `json:"display_name"`
		VerificationState string `json:"verification_state"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &statusPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if statusPayload.ID != createdPayload.ID || statusPayload.Slug != "north-studio" {
		t.Fatalf("unexpected vendor profile payload: %#v", statusPayload)
	}

	duplicate := requestJSON(t, r, http.MethodPost, "/api/v1/vendors/register", map[string]string{
		"slug":         "north-studio-two",
		"display_name": "North Studio Two",
	}, owner.AccessToken)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("expected duplicate vendor registration conflict, got status=%d body=%s", duplicate.Code, duplicate.Body.String())
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

func TestAdminPaymentSettingsRBACAndEnforcement(t *testing.T) {
	cfg := testConfig()
	cfg.Environment = "development"
	r := mustRouterWithConfig(t, cfg)

	support := registerUser(t, r, "support@example.com")
	finance := registerUser(t, r, "finance@example.com")

	supportRead := requestJSON(t, r, http.MethodGet, "/api/v1/admin/settings/payments", nil, support.AccessToken)
	if supportRead.Code != http.StatusForbidden {
		t.Fatalf("expected support to be forbidden from payment settings, got status=%d body=%s", supportRead.Code, supportRead.Body.String())
	}

	financeRead := requestJSON(t, r, http.MethodGet, "/api/v1/admin/settings/payments", nil, finance.AccessToken)
	if financeRead.Code != http.StatusOK {
		t.Fatalf("expected finance to read payment settings, got status=%d body=%s", financeRead.Code, financeRead.Body.String())
	}

	var initialSettings struct {
		StripeEnabled bool `json:"stripe_enabled"`
		CODEnabled    bool `json:"cod_enabled"`
	}
	if err := json.Unmarshal(financeRead.Body.Bytes(), &initialSettings); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !initialSettings.StripeEnabled || !initialSettings.CODEnabled {
		t.Fatalf("expected both payment methods enabled by default, got %#v", initialSettings)
	}

	disableStripe := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/settings/payments", map[string]bool{
		"stripe_enabled": false,
	}, finance.AccessToken)
	if disableStripe.Code != http.StatusOK {
		t.Fatalf("expected finance to disable stripe, got status=%d body=%s", disableStripe.Code, disableStripe.Body.String())
	}

	stripeOrderID := createGuestOrderFromSeededCatalog(t, r, "gst_settings_stripe_disabled", "idem-settings-stripe-disabled-order")
	stripeIntent := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/payments/stripe/intent", map[string]interface{}{
		"order_id":        stripeOrderID,
		"idempotency_key": "idem-settings-stripe-disabled-intent",
	}, "", map[string]string{guestTokenHeader: "gst_settings_stripe_disabled"})
	if stripeIntent.Code != http.StatusConflict {
		t.Fatalf("expected stripe intent to be blocked when disabled, got status=%d body=%s", stripeIntent.Code, stripeIntent.Body.String())
	}

	disableCOD := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/settings/payments", map[string]bool{
		"cod_enabled": false,
	}, finance.AccessToken)
	if disableCOD.Code != http.StatusOK {
		t.Fatalf("expected finance to disable cod, got status=%d body=%s", disableCOD.Code, disableCOD.Body.String())
	}

	codOrderID := createGuestOrderFromSeededCatalog(t, r, "gst_settings_cod_disabled", "idem-settings-cod-disabled-order")
	codConfirm := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/payments/cod/confirm", map[string]interface{}{
		"order_id":        codOrderID,
		"idempotency_key": "idem-settings-cod-disabled-confirm",
	}, "", map[string]string{guestTokenHeader: "gst_settings_cod_disabled"})
	if codConfirm.Code != http.StatusConflict {
		t.Fatalf("expected cod confirmation to be blocked when disabled, got status=%d body=%s", codConfirm.Code, codConfirm.Body.String())
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

func TestStripeIntentAndWebhookFlow(t *testing.T) {
	cfg := testConfig()
	cfg.Environment = "development"
	cfg.StripeWebhookSecret = "whsec_router_test"
	r := mustRouterWithConfig(t, cfg)

	catalogRes := requestJSON(t, r, http.MethodGet, "/api/v1/catalog/products", nil, "")
	if catalogRes.Code != http.StatusOK {
		t.Fatalf("catalog status=%d body=%s", catalogRes.Code, catalogRes.Body.String())
	}

	var catalogPayload struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(catalogRes.Body.Bytes(), &catalogPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(catalogPayload.Items) == 0 {
		t.Fatal("expected at least one seeded product")
	}

	guestHeaders := map[string]string{guestTokenHeader: "gst_test_stripe_flow"}
	addRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/cart/items", map[string]interface{}{
		"product_id": catalogPayload.Items[0].ID,
		"qty":        1,
	}, "", guestHeaders)
	if addRes.Code != http.StatusOK {
		t.Fatalf("add cart item status=%d body=%s", addRes.Code, addRes.Body.String())
	}

	orderRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/checkout/place-order", map[string]interface{}{
		"idempotency_key": "idem-stripe-order-1",
	}, "", guestHeaders)
	if orderRes.Code != http.StatusCreated {
		t.Fatalf("place order status=%d body=%s", orderRes.Code, orderRes.Body.String())
	}

	var orderPayload struct {
		Order struct {
			ID string `json:"id"`
		} `json:"order"`
	}
	if err := json.Unmarshal(orderRes.Body.Bytes(), &orderPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	intentRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/payments/stripe/intent", map[string]interface{}{
		"order_id":        orderPayload.Order.ID,
		"idempotency_key": "idem-stripe-intent-1",
	}, "", guestHeaders)
	if intentRes.Code != http.StatusCreated {
		t.Fatalf("create stripe intent status=%d body=%s", intentRes.Code, intentRes.Body.String())
	}

	var intentPayload struct {
		ID          string `json:"id"`
		ProviderRef string `json:"provider_ref"`
		Status      string `json:"status"`
	}
	if err := json.Unmarshal(intentRes.Body.Bytes(), &intentPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if intentPayload.Status != "pending" {
		t.Fatalf("expected pending intent status, got %s", intentPayload.Status)
	}

	webhookBody, webhookSignature := signedStripeWebhook(t, cfg.StripeWebhookSecret, "evt_router_1", "payment_intent.succeeded", intentPayload.ProviderRef)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBuffer(webhookBody))
	req.Header.Set(stripeSignatureHeader, webhookSignature)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("stripe webhook status=%d body=%s", rr.Code, rr.Body.String())
	}

	var webhookPayload struct {
		Processed bool `json:"processed"`
		Duplicate bool `json:"duplicate"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &webhookPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !webhookPayload.Processed || webhookPayload.Duplicate {
		t.Fatalf("expected webhook processed=true duplicate=false, got processed=%t duplicate=%t", webhookPayload.Processed, webhookPayload.Duplicate)
	}

	duplicateReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBuffer(webhookBody))
	duplicateReq.Header.Set(stripeSignatureHeader, webhookSignature)
	duplicateRes := httptest.NewRecorder()
	r.ServeHTTP(duplicateRes, duplicateReq)
	if duplicateRes.Code != http.StatusOK {
		t.Fatalf("duplicate webhook status=%d body=%s", duplicateRes.Code, duplicateRes.Body.String())
	}

	var duplicatePayload struct {
		Processed bool `json:"processed"`
		Duplicate bool `json:"duplicate"`
	}
	if err := json.Unmarshal(duplicateRes.Body.Bytes(), &duplicatePayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if duplicatePayload.Processed || !duplicatePayload.Duplicate {
		t.Fatalf("expected duplicate webhook processed=false duplicate=true, got processed=%t duplicate=%t", duplicatePayload.Processed, duplicatePayload.Duplicate)
	}

	orderAfterWebhook := requestJSONWithHeaders(t, r, http.MethodGet, "/api/v1/orders/"+orderPayload.Order.ID, nil, "", guestHeaders)
	if orderAfterWebhook.Code != http.StatusOK {
		t.Fatalf("get order status=%d body=%s", orderAfterWebhook.Code, orderAfterWebhook.Body.String())
	}
	var orderAfterPayload struct {
		Order struct {
			Status string `json:"status"`
		} `json:"order"`
	}
	if err := json.Unmarshal(orderAfterWebhook.Body.Bytes(), &orderAfterPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if orderAfterPayload.Order.Status != "paid" {
		t.Fatalf("expected order status paid after webhook, got %s", orderAfterPayload.Order.Status)
	}
}

func TestCODConfirmFlow(t *testing.T) {
	cfg := testConfig()
	cfg.Environment = "development"
	r := mustRouterWithConfig(t, cfg)

	catalogRes := requestJSON(t, r, http.MethodGet, "/api/v1/catalog/products", nil, "")
	if catalogRes.Code != http.StatusOK {
		t.Fatalf("catalog status=%d body=%s", catalogRes.Code, catalogRes.Body.String())
	}

	var catalogPayload struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(catalogRes.Body.Bytes(), &catalogPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(catalogPayload.Items) == 0 {
		t.Fatal("expected at least one seeded product")
	}

	guestHeaders := map[string]string{guestTokenHeader: "gst_test_cod_flow"}
	addRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/cart/items", map[string]interface{}{
		"product_id": catalogPayload.Items[0].ID,
		"qty":        1,
	}, "", guestHeaders)
	if addRes.Code != http.StatusOK {
		t.Fatalf("add cart item status=%d body=%s", addRes.Code, addRes.Body.String())
	}

	orderRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/checkout/place-order", map[string]interface{}{
		"idempotency_key": "idem-cod-order-1",
	}, "", guestHeaders)
	if orderRes.Code != http.StatusCreated {
		t.Fatalf("place order status=%d body=%s", orderRes.Code, orderRes.Body.String())
	}

	var orderPayload struct {
		Order struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"order"`
	}
	if err := json.Unmarshal(orderRes.Body.Bytes(), &orderPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if orderPayload.Order.Status != "pending_payment" {
		t.Fatalf("expected pending_payment status before cod confirmation, got %s", orderPayload.Order.Status)
	}

	codRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/payments/cod/confirm", map[string]interface{}{
		"order_id":        orderPayload.Order.ID,
		"idempotency_key": "idem-cod-confirm-1",
	}, "", guestHeaders)
	if codRes.Code != http.StatusCreated {
		t.Fatalf("cod confirm status=%d body=%s", codRes.Code, codRes.Body.String())
	}

	var codPayload struct {
		ID     string `json:"id"`
		Method string `json:"method"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(codRes.Body.Bytes(), &codPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if codPayload.Method != "cod" || codPayload.Status != "pending_collection" {
		t.Fatalf("expected cod pending_collection payment, got method=%s status=%s", codPayload.Method, codPayload.Status)
	}

	codRetryRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/payments/cod/confirm", map[string]interface{}{
		"order_id":        orderPayload.Order.ID,
		"idempotency_key": "idem-cod-confirm-1",
	}, "", guestHeaders)
	if codRetryRes.Code != http.StatusCreated {
		t.Fatalf("cod retry status=%d body=%s", codRetryRes.Code, codRetryRes.Body.String())
	}

	var codRetryPayload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(codRetryRes.Body.Bytes(), &codRetryPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if codRetryPayload.ID != codPayload.ID {
		t.Fatalf("expected idempotent cod payment id %s, got %s", codPayload.ID, codRetryPayload.ID)
	}

	orderAfterCOD := requestJSONWithHeaders(t, r, http.MethodGet, "/api/v1/orders/"+orderPayload.Order.ID, nil, "", guestHeaders)
	if orderAfterCOD.Code != http.StatusOK {
		t.Fatalf("get order status=%d body=%s", orderAfterCOD.Code, orderAfterCOD.Body.String())
	}
	var orderAfterPayload struct {
		Order struct {
			Status string `json:"status"`
		} `json:"order"`
	}
	if err := json.Unmarshal(orderAfterCOD.Body.Bytes(), &orderAfterPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if orderAfterPayload.Order.Status != "cod_confirmed" {
		t.Fatalf("expected order status cod_confirmed after cod confirmation, got %s", orderAfterPayload.Order.Status)
	}
}

func TestInvoiceDownloadRequiresConfirmedPayment(t *testing.T) {
	cfg := testConfig()
	cfg.Environment = "development"
	r := mustRouterWithConfig(t, cfg)

	catalogRes := requestJSON(t, r, http.MethodGet, "/api/v1/catalog/products", nil, "")
	if catalogRes.Code != http.StatusOK {
		t.Fatalf("catalog status=%d body=%s", catalogRes.Code, catalogRes.Body.String())
	}

	var catalogPayload struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(catalogRes.Body.Bytes(), &catalogPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(catalogPayload.Items) == 0 {
		t.Fatal("expected at least one seeded product")
	}

	guestHeaders := map[string]string{guestTokenHeader: "gst_test_invoice_flow"}
	addRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/cart/items", map[string]interface{}{
		"product_id": catalogPayload.Items[0].ID,
		"qty":        1,
	}, "", guestHeaders)
	if addRes.Code != http.StatusOK {
		t.Fatalf("add cart item status=%d body=%s", addRes.Code, addRes.Body.String())
	}

	orderRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/checkout/place-order", map[string]interface{}{
		"idempotency_key": "idem-invoice-order-1",
	}, "", guestHeaders)
	if orderRes.Code != http.StatusCreated {
		t.Fatalf("place order status=%d body=%s", orderRes.Code, orderRes.Body.String())
	}

	var orderPayload struct {
		Order struct {
			ID string `json:"id"`
		} `json:"order"`
	}
	if err := json.Unmarshal(orderRes.Body.Bytes(), &orderPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	invoiceBeforePayment := requestJSONWithHeaders(t, r, http.MethodGet, "/api/v1/invoices/"+orderPayload.Order.ID+"/download", nil, "", guestHeaders)
	if invoiceBeforePayment.Code != http.StatusConflict {
		t.Fatalf("expected invoice conflict before payment confirmation, got status=%d body=%s", invoiceBeforePayment.Code, invoiceBeforePayment.Body.String())
	}

	codRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/payments/cod/confirm", map[string]interface{}{
		"order_id":        orderPayload.Order.ID,
		"idempotency_key": "idem-invoice-cod-1",
	}, "", guestHeaders)
	if codRes.Code != http.StatusCreated {
		t.Fatalf("cod confirm status=%d body=%s", codRes.Code, codRes.Body.String())
	}

	invoiceRes := httptest.NewRecorder()
	invoiceReq := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+orderPayload.Order.ID+"/download", nil)
	invoiceReq.Header.Set(guestTokenHeader, guestHeaders[guestTokenHeader])
	r.ServeHTTP(invoiceRes, invoiceReq)
	if invoiceRes.Code != http.StatusOK {
		t.Fatalf("invoice download status=%d body=%s", invoiceRes.Code, invoiceRes.Body.String())
	}
	if got := invoiceRes.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("expected content-type application/pdf, got %s", got)
	}
	if !bytes.HasPrefix(invoiceRes.Body.Bytes(), []byte("%PDF")) {
		t.Fatalf("expected PDF payload prefix, got %q", invoiceRes.Body.String())
	}
}

func TestStripeWebhookRejectsInvalidSignature(t *testing.T) {
	cfg := testConfig()
	cfg.StripeWebhookSecret = "whsec_router_test"
	r := mustRouterWithConfig(t, cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBufferString(`{"id":"evt_invalid","type":"payment_intent.succeeded"}`))
	req.Header.Set(stripeSignatureHeader, "t=1,v1=invalid")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid signature to return 400, got status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func createGuestOrderFromSeededCatalog(t *testing.T, r http.Handler, guestToken, idempotencyKey string) string {
	t.Helper()

	catalogRes := requestJSON(t, r, http.MethodGet, "/api/v1/catalog/products", nil, "")
	if catalogRes.Code != http.StatusOK {
		t.Fatalf("catalog status=%d body=%s", catalogRes.Code, catalogRes.Body.String())
	}

	var catalogPayload struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(catalogRes.Body.Bytes(), &catalogPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(catalogPayload.Items) == 0 {
		t.Fatal("expected at least one seeded product")
	}

	guestHeaders := map[string]string{guestTokenHeader: guestToken}
	addRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/cart/items", map[string]interface{}{
		"product_id": catalogPayload.Items[0].ID,
		"qty":        1,
	}, "", guestHeaders)
	if addRes.Code != http.StatusOK {
		t.Fatalf("add cart item status=%d body=%s", addRes.Code, addRes.Body.String())
	}

	orderRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/checkout/place-order", map[string]interface{}{
		"idempotency_key": idempotencyKey,
	}, "", guestHeaders)
	if orderRes.Code != http.StatusCreated {
		t.Fatalf("place order status=%d body=%s", orderRes.Code, orderRes.Body.String())
	}

	var orderPayload struct {
		Order struct {
			ID string `json:"id"`
		} `json:"order"`
	}
	if err := json.Unmarshal(orderRes.Body.Bytes(), &orderPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return orderPayload.Order.ID
}

func signedStripeWebhook(t *testing.T, secret, eventID, eventType, paymentIntentID string) ([]byte, string) {
	t.Helper()

	payload, err := json.Marshal(map[string]interface{}{
		"id":   eventID,
		"type": eventType,
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": paymentIntentID,
			},
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload:   payload,
		Secret:    secret,
		Timestamp: time.Now().UTC(),
		Scheme:    "v1",
	})
	return signed.Payload, signed.Header
}
