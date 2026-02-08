package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
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

func TestAdminVendorVerificationQueueListAndUpdate(t *testing.T) {
	r := mustRouter(t)

	owner := registerUser(t, r, "vendor-verification-owner@example.com")
	admin := registerUser(t, r, "admin@example.com")
	support := registerUser(t, r, "support@example.com")
	finance := registerUser(t, r, "finance@example.com")

	createdVendor := requestJSON(t, r, http.MethodPost, "/api/v1/vendors/register", map[string]string{
		"slug":         "vendor-verification-queue",
		"display_name": "Vendor Verification Queue",
	}, owner.AccessToken)
	if createdVendor.Code != http.StatusCreated {
		t.Fatalf("vendor register status=%d body=%s", createdVendor.Code, createdVendor.Body.String())
	}

	var vendorPayload struct {
		ID                string `json:"id"`
		VerificationState string `json:"verification_state"`
	}
	if err := json.Unmarshal(createdVendor.Body.Bytes(), &vendorPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if vendorPayload.VerificationState != "pending" {
		t.Fatalf("expected initial pending state, got %s", vendorPayload.VerificationState)
	}

	adminList := requestJSON(t, r, http.MethodGet, "/api/v1/admin/vendors", nil, admin.AccessToken)
	if adminList.Code != http.StatusOK {
		t.Fatalf("admin vendor list status=%d body=%s", adminList.Code, adminList.Body.String())
	}
	var adminListPayload struct {
		Total int `json:"total"`
		Items []struct {
			ID                string `json:"id"`
			VerificationState string `json:"verification_state"`
		} `json:"items"`
	}
	if err := json.Unmarshal(adminList.Body.Bytes(), &adminListPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if adminListPayload.Total != 1 || len(adminListPayload.Items) != 1 {
		t.Fatalf("expected one vendor in list, got total=%d len=%d", adminListPayload.Total, len(adminListPayload.Items))
	}
	if adminListPayload.Items[0].ID != vendorPayload.ID {
		t.Fatalf("expected listed vendor id %s, got %s", vendorPayload.ID, adminListPayload.Items[0].ID)
	}

	invalidFilter := requestJSON(t, r, http.MethodGet, "/api/v1/admin/vendors?verification_state=invalid", nil, admin.AccessToken)
	if invalidFilter.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid filter to return 400, got status=%d body=%s", invalidFilter.Code, invalidFilter.Body.String())
	}

	verify := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/vendors/"+vendorPayload.ID+"/verification", map[string]string{
		"state":  "verified",
		"reason": "manual review complete",
	}, admin.AccessToken)
	if verify.Code != http.StatusOK {
		t.Fatalf("admin verify status=%d body=%s", verify.Code, verify.Body.String())
	}

	verifiedList := requestJSON(t, r, http.MethodGet, "/api/v1/admin/vendors?verification_state=verified", nil, admin.AccessToken)
	if verifiedList.Code != http.StatusOK {
		t.Fatalf("admin verified filter status=%d body=%s", verifiedList.Code, verifiedList.Body.String())
	}
	var verifiedPayload struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(verifiedList.Body.Bytes(), &verifiedPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if verifiedPayload.Total != 1 {
		t.Fatalf("expected one verified vendor, got %d", verifiedPayload.Total)
	}

	supportList := requestJSON(t, r, http.MethodGet, "/api/v1/admin/vendors", nil, support.AccessToken)
	if supportList.Code != http.StatusOK {
		t.Fatalf("support should access vendor verification queue, got status=%d body=%s", supportList.Code, supportList.Body.String())
	}

	financeList := requestJSON(t, r, http.MethodGet, "/api/v1/admin/vendors", nil, finance.AccessToken)
	if financeList.Code != http.StatusForbidden {
		t.Fatalf("finance should be forbidden for vendor verification queue, got status=%d body=%s", financeList.Code, financeList.Body.String())
	}
}

func TestAdminModerationQueueListAndDecision(t *testing.T) {
	r := mustRouter(t)

	owner := registerUser(t, r, "admin-moderation-owner@example.com")
	admin := registerUser(t, r, "admin@example.com")
	moderator := registerUser(t, r, "moderator@example.com")
	support := registerUser(t, r, "support@example.com")

	vendorCreated := requestJSON(t, r, http.MethodPost, "/api/v1/vendors/register", map[string]string{
		"slug":         "admin-moderation-vendor",
		"display_name": "Admin Moderation Vendor",
	}, owner.AccessToken)
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

	ownerLogin := loginUser(t, r, "admin-moderation-owner@example.com")
	createdProduct := requestJSON(t, r, http.MethodPost, "/api/v1/vendor/products", map[string]interface{}{
		"title":                "Moderation Queue Product",
		"description":          "Queue test product",
		"category_slug":        "stationery",
		"tags":                 []string{"queue"},
		"price_incl_tax_cents": 1800,
		"currency":             "USD",
		"stock_qty":            5,
	}, ownerLogin.AccessToken)
	if createdProduct.Code != http.StatusCreated {
		t.Fatalf("create product status=%d body=%s", createdProduct.Code, createdProduct.Body.String())
	}

	var product struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createdProduct.Body.Bytes(), &product); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	submitted := requestJSON(t, r, http.MethodPost, "/api/v1/vendor/products/"+product.ID+"/submit-moderation", map[string]string{}, ownerLogin.AccessToken)
	if submitted.Code != http.StatusOK {
		t.Fatalf("submit moderation status=%d body=%s", submitted.Code, submitted.Body.String())
	}

	pendingQueue := requestJSON(t, r, http.MethodGet, "/api/v1/admin/moderation/products?status=pending_approval", nil, moderator.AccessToken)
	if pendingQueue.Code != http.StatusOK {
		t.Fatalf("moderation queue status=%d body=%s", pendingQueue.Code, pendingQueue.Body.String())
	}
	var pendingPayload struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(pendingQueue.Body.Bytes(), &pendingPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if pendingPayload.Total != 1 {
		t.Fatalf("expected one pending moderation item, got %d", pendingPayload.Total)
	}

	invalidStatus := requestJSON(t, r, http.MethodGet, "/api/v1/admin/moderation/products?status=invalid", nil, moderator.AccessToken)
	if invalidStatus.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid status filter 400, got status=%d body=%s", invalidStatus.Code, invalidStatus.Body.String())
	}

	approved := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/moderation/products/"+product.ID, map[string]string{
		"decision": "approve",
	}, moderator.AccessToken)
	if approved.Code != http.StatusOK {
		t.Fatalf("approve moderation status=%d body=%s", approved.Code, approved.Body.String())
	}

	pendingAfterDecision := requestJSON(t, r, http.MethodGet, "/api/v1/admin/moderation/products?status=pending_approval", nil, moderator.AccessToken)
	if pendingAfterDecision.Code != http.StatusOK {
		t.Fatalf("pending queue after decision status=%d body=%s", pendingAfterDecision.Code, pendingAfterDecision.Body.String())
	}
	var pendingAfterPayload struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(pendingAfterDecision.Body.Bytes(), &pendingAfterPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if pendingAfterPayload.Total != 0 {
		t.Fatalf("expected no pending moderation items after approval, got %d", pendingAfterPayload.Total)
	}

	approvedQueue := requestJSON(t, r, http.MethodGet, "/api/v1/admin/moderation/products?status=approved", nil, moderator.AccessToken)
	if approvedQueue.Code != http.StatusOK {
		t.Fatalf("approved queue status=%d body=%s", approvedQueue.Code, approvedQueue.Body.String())
	}
	var approvedPayload struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(approvedQueue.Body.Bytes(), &approvedPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if approvedPayload.Total != 1 {
		t.Fatalf("expected one approved product in filtered queue, got %d", approvedPayload.Total)
	}

	supportList := requestJSON(t, r, http.MethodGet, "/api/v1/admin/moderation/products", nil, support.AccessToken)
	if supportList.Code != http.StatusForbidden {
		t.Fatalf("support should be forbidden for moderation queue, got status=%d body=%s", supportList.Code, supportList.Body.String())
	}
}

func TestAdminOrdersOperationsListDetailAndStatus(t *testing.T) {
	r := mustRouter(t)

	owner := registerUser(t, r, "admin-orders-owner@example.com")
	admin := registerUser(t, r, "admin@example.com")
	support := registerUser(t, r, "support@example.com")
	finance := registerUser(t, r, "finance@example.com")
	buyer := registerUser(t, r, "buyer-orders-ops@example.com")

	vendorCreated := requestJSON(t, r, http.MethodPost, "/api/v1/vendors/register", map[string]string{
		"slug":         "admin-orders-vendor",
		"display_name": "Admin Orders Vendor",
	}, owner.AccessToken)
	if vendorCreated.Code != http.StatusCreated {
		t.Fatalf("vendor register status=%d body=%s", vendorCreated.Code, vendorCreated.Body.String())
	}

	var vendorPayload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(vendorCreated.Body.Bytes(), &vendorPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	verified := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/vendors/"+vendorPayload.ID+"/verification", map[string]string{
		"state":  "verified",
		"reason": "verified for admin order operations test",
	}, admin.AccessToken)
	if verified.Code != http.StatusOK {
		t.Fatalf("vendor verify status=%d body=%s", verified.Code, verified.Body.String())
	}

	ownerLogin := loginUser(t, r, "admin-orders-owner@example.com")
	createdProduct := requestJSON(t, r, http.MethodPost, "/api/v1/vendor/products", map[string]interface{}{
		"title":                "Admin Orders Product",
		"description":          "Order operations test product",
		"category_slug":        "stationery",
		"tags":                 []string{"orders", "ops"},
		"price_incl_tax_cents": 2100,
		"currency":             "USD",
		"stock_qty":            10,
	}, ownerLogin.AccessToken)
	if createdProduct.Code != http.StatusCreated {
		t.Fatalf("create product status=%d body=%s", createdProduct.Code, createdProduct.Body.String())
	}

	var product struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createdProduct.Body.Bytes(), &product); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	submitted := requestJSON(t, r, http.MethodPost, "/api/v1/vendor/products/"+product.ID+"/submit-moderation", map[string]string{}, ownerLogin.AccessToken)
	if submitted.Code != http.StatusOK {
		t.Fatalf("submit moderation status=%d body=%s", submitted.Code, submitted.Body.String())
	}

	approved := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/moderation/products/"+product.ID, map[string]string{
		"decision": "approve",
	}, admin.AccessToken)
	if approved.Code != http.StatusOK {
		t.Fatalf("approve product status=%d body=%s", approved.Code, approved.Body.String())
	}

	guestHeaders := map[string]string{
		"X-Guest-Token": "gst-admin-orders-ops",
	}

	addToCart := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/cart/items", map[string]interface{}{
		"product_id": product.ID,
		"qty":        1,
	}, "", guestHeaders)
	if addToCart.Code != http.StatusOK {
		t.Fatalf("add cart item status=%d body=%s", addToCart.Code, addToCart.Body.String())
	}

	placeOrder := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/checkout/place-order", map[string]interface{}{
		"idempotency_key": "idem-admin-orders-ops-order-1",
	}, "", guestHeaders)
	if placeOrder.Code != http.StatusCreated {
		t.Fatalf("place order status=%d body=%s", placeOrder.Code, placeOrder.Body.String())
	}

	var orderPayload struct {
		Order struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"order"`
	}
	if err := json.Unmarshal(placeOrder.Body.Bytes(), &orderPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if orderPayload.Order.Status != "pending_payment" {
		t.Fatalf("expected pending_payment order status, got %s", orderPayload.Order.Status)
	}

	supportOrders := requestJSON(t, r, http.MethodGet, "/api/v1/admin/orders", nil, support.AccessToken)
	if supportOrders.Code != http.StatusOK {
		t.Fatalf("support list orders status=%d body=%s", supportOrders.Code, supportOrders.Body.String())
	}
	var supportOrdersPayload struct {
		Total int `json:"total"`
		Items []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(supportOrders.Body.Bytes(), &supportOrdersPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if supportOrdersPayload.Total != 1 || len(supportOrdersPayload.Items) != 1 {
		t.Fatalf("expected one order in support list, got total=%d len=%d", supportOrdersPayload.Total, len(supportOrdersPayload.Items))
	}
	if supportOrdersPayload.Items[0].ID != orderPayload.Order.ID {
		t.Fatalf("expected order id %s, got %s", orderPayload.Order.ID, supportOrdersPayload.Items[0].ID)
	}

	pendingFilter := requestJSON(t, r, http.MethodGet, "/api/v1/admin/orders?status=pending_payment", nil, support.AccessToken)
	if pendingFilter.Code != http.StatusOK {
		t.Fatalf("pending filter status=%d body=%s", pendingFilter.Code, pendingFilter.Body.String())
	}

	invalidFilter := requestJSON(t, r, http.MethodGet, "/api/v1/admin/orders?status=invalid", nil, support.AccessToken)
	if invalidFilter.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid status filter 400, got status=%d body=%s", invalidFilter.Code, invalidFilter.Body.String())
	}

	orderDetail := requestJSON(t, r, http.MethodGet, "/api/v1/admin/orders/"+orderPayload.Order.ID, nil, support.AccessToken)
	if orderDetail.Code != http.StatusOK {
		t.Fatalf("order detail status=%d body=%s", orderDetail.Code, orderDetail.Body.String())
	}

	financeOrders := requestJSON(t, r, http.MethodGet, "/api/v1/admin/orders", nil, finance.AccessToken)
	if financeOrders.Code != http.StatusForbidden {
		t.Fatalf("finance should be forbidden for admin orders ops, got status=%d body=%s", financeOrders.Code, financeOrders.Body.String())
	}

	buyerOrders := requestJSON(t, r, http.MethodGet, "/api/v1/admin/orders", nil, buyer.AccessToken)
	if buyerOrders.Code != http.StatusForbidden {
		t.Fatalf("buyer should be forbidden for admin orders ops, got status=%d body=%s", buyerOrders.Code, buyerOrders.Body.String())
	}

	invalidStatus := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/orders/"+orderPayload.Order.ID+"/status", map[string]string{
		"status": "invalid",
	}, support.AccessToken)
	if invalidStatus.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid status update 400, got status=%d body=%s", invalidStatus.Code, invalidStatus.Body.String())
	}

	markedFailed := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/orders/"+orderPayload.Order.ID+"/status", map[string]string{
		"status": "payment_failed",
	}, support.AccessToken)
	if markedFailed.Code != http.StatusOK {
		t.Fatalf("mark payment_failed status=%d body=%s", markedFailed.Code, markedFailed.Body.String())
	}

	markedPaid := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/orders/"+orderPayload.Order.ID+"/status", map[string]string{
		"status": "paid",
	}, support.AccessToken)
	if markedPaid.Code != http.StatusOK {
		t.Fatalf("mark paid status=%d body=%s", markedPaid.Code, markedPaid.Body.String())
	}

	revertFromPaid := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/orders/"+orderPayload.Order.ID+"/status", map[string]string{
		"status": "payment_failed",
	}, support.AccessToken)
	if revertFromPaid.Code != http.StatusConflict {
		t.Fatalf("expected paid->payment_failed transition conflict, got status=%d body=%s", revertFromPaid.Code, revertFromPaid.Body.String())
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

func TestAdminPromotionsCRUDAndRBAC(t *testing.T) {
	r := mustRouter(t)

	support := registerUser(t, r, "support@example.com")
	finance := registerUser(t, r, "finance@example.com")
	buyer := registerUser(t, r, "buyer-promotions@example.com")

	supportList := requestJSON(t, r, http.MethodGet, "/api/v1/admin/promotions", nil, support.AccessToken)
	if supportList.Code != http.StatusForbidden {
		t.Fatalf("expected support forbidden for promotions list, got status=%d body=%s", supportList.Code, supportList.Body.String())
	}
	buyerList := requestJSON(t, r, http.MethodGet, "/api/v1/admin/promotions", nil, buyer.AccessToken)
	if buyerList.Code != http.StatusForbidden {
		t.Fatalf("expected buyer forbidden for promotions list, got status=%d body=%s", buyerList.Code, buyerList.Body.String())
	}

	invalidCreate := requestJSON(t, r, http.MethodPost, "/api/v1/admin/promotions", map[string]interface{}{
		"name":      "Broken Promotion",
		"rule_json": "not-an-object",
	}, finance.AccessToken)
	if invalidCreate.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid promotion payload 400, got status=%d body=%s", invalidCreate.Code, invalidCreate.Body.String())
	}

	created := requestJSON(t, r, http.MethodPost, "/api/v1/admin/promotions", map[string]interface{}{
		"name":      "Platform Spring Sale",
		"rule_json": map[string]interface{}{"type": "percentage", "value": 12},
		"active":    true,
		"stackable": false,
	}, finance.AccessToken)
	if created.Code != http.StatusCreated {
		t.Fatalf("expected finance to create promotion, got status=%d body=%s", created.Code, created.Body.String())
	}

	var createdPayload struct {
		ID       string                 `json:"id"`
		Name     string                 `json:"name"`
		RuleJSON map[string]interface{} `json:"rule_json"`
		Active   bool                   `json:"active"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &createdPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if createdPayload.ID == "" {
		t.Fatalf("expected created promotion id, got %#v", createdPayload)
	}

	list := requestJSON(t, r, http.MethodGet, "/api/v1/admin/promotions", nil, finance.AccessToken)
	if list.Code != http.StatusOK {
		t.Fatalf("expected finance list promotions 200, got status=%d body=%s", list.Code, list.Body.String())
	}
	var listPayload struct {
		Total int `json:"total"`
		Items []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if listPayload.Total != 1 || len(listPayload.Items) != 1 {
		t.Fatalf("expected one promotion listed, got total=%d len=%d", listPayload.Total, len(listPayload.Items))
	}
	if listPayload.Items[0].ID != createdPayload.ID {
		t.Fatalf("expected promotion id %s in list, got %s", createdPayload.ID, listPayload.Items[0].ID)
	}

	invalidUpdate := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/promotions/"+createdPayload.ID, map[string]interface{}{}, finance.AccessToken)
	if invalidUpdate.Code != http.StatusBadRequest {
		t.Fatalf("expected empty update payload 400, got status=%d body=%s", invalidUpdate.Code, invalidUpdate.Body.String())
	}

	updated := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/promotions/"+createdPayload.ID, map[string]interface{}{
		"name":      "Platform Spring Sale Updated",
		"rule_json": map[string]interface{}{"type": "fixed", "value": 500},
		"active":    false,
		"stackable": true,
	}, finance.AccessToken)
	if updated.Code != http.StatusOK {
		t.Fatalf("expected update promotion 200, got status=%d body=%s", updated.Code, updated.Body.String())
	}

	deleted := requestJSON(t, r, http.MethodDelete, "/api/v1/admin/promotions/"+createdPayload.ID, nil, finance.AccessToken)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("expected delete promotion 204, got status=%d body=%s", deleted.Code, deleted.Body.String())
	}

	afterDelete := requestJSON(t, r, http.MethodGet, "/api/v1/admin/promotions", nil, finance.AccessToken)
	if afterDelete.Code != http.StatusOK {
		t.Fatalf("expected list promotions after delete 200, got status=%d body=%s", afterDelete.Code, afterDelete.Body.String())
	}
	var afterDeletePayload struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(afterDelete.Body.Bytes(), &afterDeletePayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if afterDeletePayload.Total != 0 {
		t.Fatalf("expected zero promotions after delete, got %d", afterDeletePayload.Total)
	}
}

func TestAdminAuditLogsListAndRBAC(t *testing.T) {
	r := mustRouter(t)

	support := registerUser(t, r, "support@example.com")
	finance := registerUser(t, r, "finance@example.com")
	buyer := registerUser(t, r, "buyer-audit@example.com")

	vendorCreated := requestJSON(t, r, http.MethodPost, "/api/v1/vendors/register", map[string]string{
		"slug":         "vendor-audit",
		"display_name": "Vendor Audit",
	}, buyer.AccessToken)
	if vendorCreated.Code != http.StatusCreated {
		t.Fatalf("vendor register status=%d body=%s", vendorCreated.Code, vendorCreated.Body.String())
	}
	var vendor struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(vendorCreated.Body.Bytes(), &vendor); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	verified := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/vendors/"+vendor.ID+"/verification", map[string]string{
		"state":  "verified",
		"reason": "kyc complete",
	}, support.AccessToken)
	if verified.Code != http.StatusOK {
		t.Fatalf("admin verify vendor status=%d body=%s", verified.Code, verified.Body.String())
	}

	supportList := requestJSON(t, r, http.MethodGet, "/api/v1/admin/audit-logs", nil, support.AccessToken)
	if supportList.Code != http.StatusOK {
		t.Fatalf("support audit log list status=%d body=%s", supportList.Code, supportList.Body.String())
	}
	var supportPayload struct {
		Total int `json:"total"`
		Items []struct {
			ActorID    string `json:"actor_id"`
			ActorType  string `json:"actor_type"`
			Action     string `json:"action"`
			TargetType string `json:"target_type"`
			TargetID   string `json:"target_id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(supportList.Body.Bytes(), &supportPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if supportPayload.Total < 1 || len(supportPayload.Items) < 1 {
		t.Fatalf("expected at least one audit log item, got total=%d len=%d", supportPayload.Total, len(supportPayload.Items))
	}
	if supportPayload.Items[0].ActorID != support.User.ID {
		t.Fatalf("expected actor_id %s, got %s", support.User.ID, supportPayload.Items[0].ActorID)
	}
	if supportPayload.Items[0].ActorType != "admin" {
		t.Fatalf("expected actor_type admin, got %s", supportPayload.Items[0].ActorType)
	}
	if supportPayload.Items[0].Action != "vendor_verification_updated" {
		t.Fatalf("expected action vendor_verification_updated, got %s", supportPayload.Items[0].Action)
	}
	if supportPayload.Items[0].TargetType != "vendor" {
		t.Fatalf("expected target_type vendor, got %s", supportPayload.Items[0].TargetType)
	}
	if supportPayload.Items[0].TargetID != vendor.ID {
		t.Fatalf("expected target_id %s, got %s", vendor.ID, supportPayload.Items[0].TargetID)
	}

	filtered := requestJSON(
		t,
		r,
		http.MethodGet,
		"/api/v1/admin/audit-logs?action=vendor_verification_updated&target_type=vendor&target_id="+vendor.ID,
		nil,
		support.AccessToken,
	)
	if filtered.Code != http.StatusOK {
		t.Fatalf("filtered audit logs status=%d body=%s", filtered.Code, filtered.Body.String())
	}
	var filteredPayload struct {
		Total int `json:"total"`
		Items []struct {
			TargetID string `json:"target_id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(filtered.Body.Bytes(), &filteredPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if filteredPayload.Total != 1 || len(filteredPayload.Items) != 1 {
		t.Fatalf("expected one filtered audit log item, got total=%d len=%d", filteredPayload.Total, len(filteredPayload.Items))
	}
	if filteredPayload.Items[0].TargetID != vendor.ID {
		t.Fatalf("expected filtered target_id %s, got %s", vendor.ID, filteredPayload.Items[0].TargetID)
	}

	financeList := requestJSON(t, r, http.MethodGet, "/api/v1/admin/audit-logs?limit=1", nil, finance.AccessToken)
	if financeList.Code != http.StatusOK {
		t.Fatalf("finance audit log list status=%d body=%s", financeList.Code, financeList.Body.String())
	}

	buyerList := requestJSON(t, r, http.MethodGet, "/api/v1/admin/audit-logs", nil, buyer.AccessToken)
	if buyerList.Code != http.StatusForbidden {
		t.Fatalf("buyer should be forbidden from audit logs, got status=%d body=%s", buyerList.Code, buyerList.Body.String())
	}

	invalidLimit := requestJSON(t, r, http.MethodGet, "/api/v1/admin/audit-logs?limit=0", nil, support.AccessToken)
	if invalidLimit.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid limit to fail, got status=%d body=%s", invalidLimit.Code, invalidLimit.Body.String())
	}

	invalidOffset := requestJSON(t, r, http.MethodGet, "/api/v1/admin/audit-logs?offset=-1", nil, support.AccessToken)
	if invalidOffset.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid offset to fail, got status=%d body=%s", invalidOffset.Code, invalidOffset.Body.String())
	}
}

func TestBuyerPaymentSettingsExposeCurrentAvailability(t *testing.T) {
	cfg := testConfig()
	cfg.Environment = "development"
	r := mustRouterWithConfig(t, cfg)

	finance := registerUser(t, r, "finance@example.com")

	initialSettings := requestJSON(t, r, http.MethodGet, "/api/v1/payments/settings", nil, "")
	if initialSettings.Code != http.StatusOK {
		t.Fatalf("expected buyer payment settings to be readable, got status=%d body=%s", initialSettings.Code, initialSettings.Body.String())
	}

	var initialPayload struct {
		StripeEnabled bool `json:"stripe_enabled"`
		CODEnabled    bool `json:"cod_enabled"`
	}
	if err := json.Unmarshal(initialSettings.Body.Bytes(), &initialPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !initialPayload.StripeEnabled || !initialPayload.CODEnabled {
		t.Fatalf("expected both methods enabled by default, got %#v", initialPayload)
	}

	guestToken := initialSettings.Header().Get(guestTokenHeader)
	if strings.TrimSpace(guestToken) == "" {
		t.Fatalf("expected buyer settings response to include guest token header")
	}

	disableStripe := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/settings/payments", map[string]bool{
		"stripe_enabled": false,
	}, finance.AccessToken)
	if disableStripe.Code != http.StatusOK {
		t.Fatalf("expected finance to disable stripe, got status=%d body=%s", disableStripe.Code, disableStripe.Body.String())
	}

	afterDisable := requestJSONWithHeaders(t, r, http.MethodGet, "/api/v1/payments/settings", nil, "", map[string]string{
		guestTokenHeader: guestToken,
	})
	if afterDisable.Code != http.StatusOK {
		t.Fatalf("expected buyer payment settings after update, got status=%d body=%s", afterDisable.Code, afterDisable.Body.String())
	}

	var afterPayload struct {
		StripeEnabled bool `json:"stripe_enabled"`
		CODEnabled    bool `json:"cod_enabled"`
	}
	if err := json.Unmarshal(afterDisable.Body.Bytes(), &afterPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if afterPayload.StripeEnabled {
		t.Fatalf("expected stripe to be disabled in buyer settings, got %#v", afterPayload)
	}
	if !afterPayload.CODEnabled {
		t.Fatalf("expected cod to remain enabled in buyer settings, got %#v", afterPayload)
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

func TestVendorProductsAndCouponsCRUD(t *testing.T) {
	r := mustRouter(t)

	owner := registerUser(t, r, "vendor-products-owner@example.com")
	admin := registerUser(t, r, "admin@example.com")
	moderator := registerUser(t, r, "moderator@example.com")
	buyer := registerUser(t, r, "buyer-products@example.com")

	vendorCreated := requestJSON(t, r, http.MethodPost, "/api/v1/vendors/register", map[string]string{
		"slug":         "vendor-products",
		"display_name": "Vendor Products",
	}, owner.AccessToken)
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

	ownerLogin := loginUser(t, r, "vendor-products-owner@example.com")

	createdProduct := requestJSON(t, r, http.MethodPost, "/api/v1/vendor/products", map[string]interface{}{
		"title":                "Field Notebook",
		"description":          "Rugged paper notebook",
		"category_slug":        "stationery",
		"tags":                 []string{"paper", "field"},
		"price_incl_tax_cents": 2899,
		"currency":             "USD",
		"stock_qty":            12,
	}, ownerLogin.AccessToken)
	if createdProduct.Code != http.StatusCreated {
		t.Fatalf("create product status=%d body=%s", createdProduct.Code, createdProduct.Body.String())
	}

	var product struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(createdProduct.Body.Bytes(), &product); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if product.Status != "draft" {
		t.Fatalf("expected draft product status, got %s", product.Status)
	}

	listProducts := requestJSON(t, r, http.MethodGet, "/api/v1/vendor/products", nil, ownerLogin.AccessToken)
	if listProducts.Code != http.StatusOK {
		t.Fatalf("list products status=%d body=%s", listProducts.Code, listProducts.Body.String())
	}
	var productListPayload struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(listProducts.Body.Bytes(), &productListPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if productListPayload.Total != 1 {
		t.Fatalf("expected one vendor product, got %d", productListPayload.Total)
	}

	updatedProduct := requestJSON(t, r, http.MethodPatch, "/api/v1/vendor/products/"+product.ID, map[string]interface{}{
		"title":                "Field Notebook Pro",
		"price_incl_tax_cents": 3499,
	}, ownerLogin.AccessToken)
	if updatedProduct.Code != http.StatusOK {
		t.Fatalf("update product status=%d body=%s", updatedProduct.Code, updatedProduct.Body.String())
	}

	submitted := requestJSON(t, r, http.MethodPost, "/api/v1/vendor/products/"+product.ID+"/submit-moderation", map[string]string{}, ownerLogin.AccessToken)
	if submitted.Code != http.StatusOK {
		t.Fatalf("submit moderation status=%d body=%s", submitted.Code, submitted.Body.String())
	}

	approved := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/moderation/products/"+product.ID, map[string]string{
		"decision": "approve",
	}, moderator.AccessToken)
	if approved.Code != http.StatusOK {
		t.Fatalf("approve moderation status=%d body=%s", approved.Code, approved.Body.String())
	}

	revisedApproved := requestJSON(t, r, http.MethodPatch, "/api/v1/vendor/products/"+product.ID, map[string]interface{}{
		"price_incl_tax_cents": 3799,
	}, ownerLogin.AccessToken)
	if revisedApproved.Code != http.StatusOK {
		t.Fatalf("revise approved product status=%d body=%s", revisedApproved.Code, revisedApproved.Body.String())
	}
	var revisedProduct struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(revisedApproved.Body.Bytes(), &revisedProduct); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if revisedProduct.Status != "draft" {
		t.Fatalf("expected approved edits to move product back to draft, got %s", revisedProduct.Status)
	}

	deleteProduct := requestJSON(t, r, http.MethodDelete, "/api/v1/vendor/products/"+product.ID, nil, ownerLogin.AccessToken)
	if deleteProduct.Code != http.StatusNoContent {
		t.Fatalf("delete product status=%d body=%s", deleteProduct.Code, deleteProduct.Body.String())
	}

	createCoupon := requestJSON(t, r, http.MethodPost, "/api/v1/vendor/coupons", map[string]interface{}{
		"code":           "save10",
		"discount_type":  "percent",
		"discount_value": 10,
	}, ownerLogin.AccessToken)
	if createCoupon.Code != http.StatusCreated {
		t.Fatalf("create coupon status=%d body=%s", createCoupon.Code, createCoupon.Body.String())
	}
	var coupon struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createCoupon.Body.Bytes(), &coupon); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	listCoupons := requestJSON(t, r, http.MethodGet, "/api/v1/vendor/coupons", nil, ownerLogin.AccessToken)
	if listCoupons.Code != http.StatusOK {
		t.Fatalf("list coupons status=%d body=%s", listCoupons.Code, listCoupons.Body.String())
	}
	var couponsPayload struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(listCoupons.Body.Bytes(), &couponsPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if couponsPayload.Total != 1 {
		t.Fatalf("expected one coupon, got %d", couponsPayload.Total)
	}

	updateCoupon := requestJSON(t, r, http.MethodPatch, "/api/v1/vendor/coupons/"+coupon.ID, map[string]interface{}{
		"active": false,
	}, ownerLogin.AccessToken)
	if updateCoupon.Code != http.StatusOK {
		t.Fatalf("update coupon status=%d body=%s", updateCoupon.Code, updateCoupon.Body.String())
	}

	deleteCoupon := requestJSON(t, r, http.MethodDelete, "/api/v1/vendor/coupons/"+coupon.ID, nil, ownerLogin.AccessToken)
	if deleteCoupon.Code != http.StatusNoContent {
		t.Fatalf("delete coupon status=%d body=%s", deleteCoupon.Code, deleteCoupon.Body.String())
	}

	couponForbidden := requestJSON(t, r, http.MethodGet, "/api/v1/vendor/coupons", nil, buyer.AccessToken)
	if couponForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected buyer to be forbidden for vendor coupons, got status=%d body=%s", couponForbidden.Code, couponForbidden.Body.String())
	}
}

func TestVendorShipmentsListDetailAndStatusUpdate(t *testing.T) {
	r := mustRouter(t)

	owner := registerUser(t, r, "vendor-shipments-owner@example.com")
	admin := registerUser(t, r, "admin@example.com")
	moderator := registerUser(t, r, "moderator@example.com")
	buyer := registerUser(t, r, "buyer-shipments@example.com")

	vendorCreated := requestJSON(t, r, http.MethodPost, "/api/v1/vendors/register", map[string]string{
		"slug":         "vendor-shipments",
		"display_name": "Vendor Shipments",
	}, owner.AccessToken)
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

	ownerLogin := loginUser(t, r, "vendor-shipments-owner@example.com")
	createdProduct := requestJSON(t, r, http.MethodPost, "/api/v1/vendor/products", map[string]interface{}{
		"title":                "Shipment Product",
		"description":          "Product to test shipment flow",
		"category_slug":        "stationery",
		"tags":                 []string{"ship"},
		"price_incl_tax_cents": 2400,
		"currency":             "USD",
		"stock_qty":            8,
	}, ownerLogin.AccessToken)
	if createdProduct.Code != http.StatusCreated {
		t.Fatalf("create product status=%d body=%s", createdProduct.Code, createdProduct.Body.String())
	}

	var product struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createdProduct.Body.Bytes(), &product); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	submitted := requestJSON(t, r, http.MethodPost, "/api/v1/vendor/products/"+product.ID+"/submit-moderation", map[string]string{}, ownerLogin.AccessToken)
	if submitted.Code != http.StatusOK {
		t.Fatalf("submit moderation status=%d body=%s", submitted.Code, submitted.Body.String())
	}

	approved := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/moderation/products/"+product.ID, map[string]string{
		"decision": "approve",
	}, moderator.AccessToken)
	if approved.Code != http.StatusOK {
		t.Fatalf("approve moderation status=%d body=%s", approved.Code, approved.Body.String())
	}

	guestHeaders := map[string]string{guestTokenHeader: "gst_vendor_shipments_flow"}
	addRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/cart/items", map[string]interface{}{
		"product_id": product.ID,
		"qty":        1,
	}, "", guestHeaders)
	if addRes.Code != http.StatusOK {
		t.Fatalf("add cart item status=%d body=%s", addRes.Code, addRes.Body.String())
	}

	orderRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/checkout/place-order", map[string]interface{}{
		"idempotency_key": "idem-vendor-shipments-order-1",
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

	codRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/payments/cod/confirm", map[string]interface{}{
		"order_id":        orderPayload.Order.ID,
		"idempotency_key": "idem-vendor-shipments-cod-1",
	}, "", guestHeaders)
	if codRes.Code != http.StatusCreated {
		t.Fatalf("cod confirm status=%d body=%s", codRes.Code, codRes.Body.String())
	}

	listRes := requestJSON(t, r, http.MethodGet, "/api/v1/vendor/shipments", nil, ownerLogin.AccessToken)
	if listRes.Code != http.StatusOK {
		t.Fatalf("list shipments status=%d body=%s", listRes.Code, listRes.Body.String())
	}

	var listPayload struct {
		Total int `json:"total"`
		Items []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listRes.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if listPayload.Total != 1 || len(listPayload.Items) != 1 {
		t.Fatalf("expected one shipment, got total=%d len=%d", listPayload.Total, len(listPayload.Items))
	}
	if listPayload.Items[0].Status != "pending" {
		t.Fatalf("expected pending shipment status, got %s", listPayload.Items[0].Status)
	}

	shipmentID := listPayload.Items[0].ID
	detailRes := requestJSON(t, r, http.MethodGet, "/api/v1/vendor/shipments/"+shipmentID, nil, ownerLogin.AccessToken)
	if detailRes.Code != http.StatusOK {
		t.Fatalf("shipment detail status=%d body=%s", detailRes.Code, detailRes.Body.String())
	}

	updatePacked := requestJSON(t, r, http.MethodPatch, "/api/v1/vendor/shipments/"+shipmentID+"/status", map[string]string{
		"status": "packed",
	}, ownerLogin.AccessToken)
	if updatePacked.Code != http.StatusOK {
		t.Fatalf("shipment packed status=%d body=%s", updatePacked.Code, updatePacked.Body.String())
	}

	updateShipped := requestJSON(t, r, http.MethodPatch, "/api/v1/vendor/shipments/"+shipmentID+"/status", map[string]string{
		"status": "shipped",
	}, ownerLogin.AccessToken)
	if updateShipped.Code != http.StatusOK {
		t.Fatalf("shipment shipped status=%d body=%s", updateShipped.Code, updateShipped.Body.String())
	}

	invalidTransition := requestJSON(t, r, http.MethodPatch, "/api/v1/vendor/shipments/"+shipmentID+"/status", map[string]string{
		"status": "pending",
	}, ownerLogin.AccessToken)
	if invalidTransition.Code != http.StatusConflict {
		t.Fatalf("expected invalid transition conflict, got status=%d body=%s", invalidTransition.Code, invalidTransition.Body.String())
	}

	forbiddenBuyer := requestJSON(t, r, http.MethodGet, "/api/v1/vendor/shipments", nil, buyer.AccessToken)
	if forbiddenBuyer.Code != http.StatusForbidden {
		t.Fatalf("expected buyer forbidden for vendor shipments, got status=%d body=%s", forbiddenBuyer.Code, forbiddenBuyer.Body.String())
	}
}

func TestVendorRefundRequestsAndDecisionFlow(t *testing.T) {
	r := mustRouter(t)

	owner := registerUser(t, r, "vendor-refunds-owner@example.com")
	admin := registerUser(t, r, "admin@example.com")
	moderator := registerUser(t, r, "moderator@example.com")
	buyer := registerUser(t, r, "buyer-refunds@example.com")

	vendorCreated := requestJSON(t, r, http.MethodPost, "/api/v1/vendors/register", map[string]string{
		"slug":         "vendor-refunds",
		"display_name": "Vendor Refunds",
	}, owner.AccessToken)
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

	ownerLogin := loginUser(t, r, "vendor-refunds-owner@example.com")
	createdProduct := requestJSON(t, r, http.MethodPost, "/api/v1/vendor/products", map[string]interface{}{
		"title":                "Refund Product",
		"description":          "Product to test vendor refund queue",
		"category_slug":        "stationery",
		"tags":                 []string{"refund"},
		"price_incl_tax_cents": 2500,
		"currency":             "USD",
		"stock_qty":            8,
	}, ownerLogin.AccessToken)
	if createdProduct.Code != http.StatusCreated {
		t.Fatalf("create product status=%d body=%s", createdProduct.Code, createdProduct.Body.String())
	}

	var product struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createdProduct.Body.Bytes(), &product); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	submitted := requestJSON(t, r, http.MethodPost, "/api/v1/vendor/products/"+product.ID+"/submit-moderation", map[string]string{}, ownerLogin.AccessToken)
	if submitted.Code != http.StatusOK {
		t.Fatalf("submit moderation status=%d body=%s", submitted.Code, submitted.Body.String())
	}

	approved := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/moderation/products/"+product.ID, map[string]string{
		"decision": "approve",
	}, moderator.AccessToken)
	if approved.Code != http.StatusOK {
		t.Fatalf("approve moderation status=%d body=%s", approved.Code, approved.Body.String())
	}

	guestHeaders := map[string]string{guestTokenHeader: "gst_vendor_refunds_flow"}
	addRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/cart/items", map[string]interface{}{
		"product_id": product.ID,
		"qty":        1,
	}, "", guestHeaders)
	if addRes.Code != http.StatusOK {
		t.Fatalf("add cart item status=%d body=%s", addRes.Code, addRes.Body.String())
	}

	orderRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/checkout/place-order", map[string]interface{}{
		"idempotency_key": "idem-vendor-refunds-order-1",
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

	codRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/payments/cod/confirm", map[string]interface{}{
		"order_id":        orderPayload.Order.ID,
		"idempotency_key": "idem-vendor-refunds-cod-1",
	}, "", guestHeaders)
	if codRes.Code != http.StatusCreated {
		t.Fatalf("cod confirm status=%d body=%s", codRes.Code, codRes.Body.String())
	}

	shipmentListRes := requestJSON(t, r, http.MethodGet, "/api/v1/vendor/shipments", nil, ownerLogin.AccessToken)
	if shipmentListRes.Code != http.StatusOK {
		t.Fatalf("list shipments status=%d body=%s", shipmentListRes.Code, shipmentListRes.Body.String())
	}

	var shipmentListPayload struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(shipmentListRes.Body.Bytes(), &shipmentListPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(shipmentListPayload.Items) != 1 {
		t.Fatalf("expected one shipment, got len=%d", len(shipmentListPayload.Items))
	}

	refundCreateRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/orders/"+orderPayload.Order.ID+"/refund-requests", map[string]interface{}{
		"shipment_id": shipmentListPayload.Items[0].ID,
		"reason":      "Item arrived damaged",
	}, "", guestHeaders)
	if refundCreateRes.Code != http.StatusCreated {
		t.Fatalf("create refund request status=%d body=%s", refundCreateRes.Code, refundCreateRes.Body.String())
	}

	var refundCreatePayload struct {
		RefundRequest struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"refund_request"`
	}
	if err := json.Unmarshal(refundCreateRes.Body.Bytes(), &refundCreatePayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if refundCreatePayload.RefundRequest.Status != "pending" {
		t.Fatalf("expected pending refund status, got %s", refundCreatePayload.RefundRequest.Status)
	}

	vendorRefundsRes := requestJSON(t, r, http.MethodGet, "/api/v1/vendor/refund-requests", nil, ownerLogin.AccessToken)
	if vendorRefundsRes.Code != http.StatusOK {
		t.Fatalf("vendor list refund requests status=%d body=%s", vendorRefundsRes.Code, vendorRefundsRes.Body.String())
	}

	var vendorRefundsPayload struct {
		Total int `json:"total"`
		Items []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(vendorRefundsRes.Body.Bytes(), &vendorRefundsPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if vendorRefundsPayload.Total != 1 || len(vendorRefundsPayload.Items) != 1 {
		t.Fatalf("expected one vendor refund request, got total=%d len=%d", vendorRefundsPayload.Total, len(vendorRefundsPayload.Items))
	}

	refundID := vendorRefundsPayload.Items[0].ID
	approveRes := requestJSON(t, r, http.MethodPatch, "/api/v1/vendor/refund-requests/"+refundID+"/decision", map[string]string{
		"decision":        "approve",
		"decision_reason": "Approved after evidence review",
	}, ownerLogin.AccessToken)
	if approveRes.Code != http.StatusOK {
		t.Fatalf("approve refund status=%d body=%s", approveRes.Code, approveRes.Body.String())
	}

	duplicateDecisionRes := requestJSON(t, r, http.MethodPatch, "/api/v1/vendor/refund-requests/"+refundID+"/decision", map[string]string{
		"decision": "reject",
	}, ownerLogin.AccessToken)
	if duplicateDecisionRes.Code != http.StatusConflict {
		t.Fatalf("expected conflict for duplicate decision, got status=%d body=%s", duplicateDecisionRes.Code, duplicateDecisionRes.Body.String())
	}

	buyerVendorListRes := requestJSON(t, r, http.MethodGet, "/api/v1/vendor/refund-requests", nil, buyer.AccessToken)
	if buyerVendorListRes.Code != http.StatusForbidden {
		t.Fatalf("expected buyer forbidden for vendor refund list, got status=%d body=%s", buyerVendorListRes.Code, buyerVendorListRes.Body.String())
	}
}

func TestVendorAnalyticsOverviewTopProductsAndCoupons(t *testing.T) {
	r := mustRouter(t)

	owner := registerUser(t, r, "vendor-analytics-owner@example.com")
	admin := registerUser(t, r, "admin@example.com")
	moderator := registerUser(t, r, "moderator@example.com")
	buyer := registerUser(t, r, "buyer-analytics@example.com")

	vendorCreated := requestJSON(t, r, http.MethodPost, "/api/v1/vendors/register", map[string]string{
		"slug":         "vendor-analytics",
		"display_name": "Vendor Analytics",
	}, owner.AccessToken)
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

	ownerLogin := loginUser(t, r, "vendor-analytics-owner@example.com")

	createdProduct := requestJSON(t, r, http.MethodPost, "/api/v1/vendor/products", map[string]interface{}{
		"title":                "Analytics Product",
		"description":          "Product to test vendor analytics",
		"category_slug":        "stationery",
		"tags":                 []string{"analytics"},
		"price_incl_tax_cents": 3100,
		"currency":             "USD",
		"stock_qty":            15,
	}, ownerLogin.AccessToken)
	if createdProduct.Code != http.StatusCreated {
		t.Fatalf("create product status=%d body=%s", createdProduct.Code, createdProduct.Body.String())
	}

	var product struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createdProduct.Body.Bytes(), &product); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	submitted := requestJSON(t, r, http.MethodPost, "/api/v1/vendor/products/"+product.ID+"/submit-moderation", map[string]string{}, ownerLogin.AccessToken)
	if submitted.Code != http.StatusOK {
		t.Fatalf("submit moderation status=%d body=%s", submitted.Code, submitted.Body.String())
	}

	approved := requestJSON(t, r, http.MethodPatch, "/api/v1/admin/moderation/products/"+product.ID, map[string]string{
		"decision": "approve",
	}, moderator.AccessToken)
	if approved.Code != http.StatusOK {
		t.Fatalf("approve moderation status=%d body=%s", approved.Code, approved.Body.String())
	}

	createCoupon := requestJSON(t, r, http.MethodPost, "/api/v1/vendor/coupons", map[string]interface{}{
		"code":           "ANALYTICS10",
		"discount_type":  "percent",
		"discount_value": 10,
	}, ownerLogin.AccessToken)
	if createCoupon.Code != http.StatusCreated {
		t.Fatalf("create coupon status=%d body=%s", createCoupon.Code, createCoupon.Body.String())
	}

	guestHeaders := map[string]string{guestTokenHeader: "gst_vendor_analytics_flow"}
	addRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/cart/items", map[string]interface{}{
		"product_id": product.ID,
		"qty":        2,
	}, "", guestHeaders)
	if addRes.Code != http.StatusOK {
		t.Fatalf("add cart item status=%d body=%s", addRes.Code, addRes.Body.String())
	}

	orderRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/checkout/place-order", map[string]interface{}{
		"idempotency_key": "idem-vendor-analytics-order-1",
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

	codRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/payments/cod/confirm", map[string]interface{}{
		"order_id":        orderPayload.Order.ID,
		"idempotency_key": "idem-vendor-analytics-cod-1",
	}, "", guestHeaders)
	if codRes.Code != http.StatusCreated {
		t.Fatalf("cod confirm status=%d body=%s", codRes.Code, codRes.Body.String())
	}

	shipmentListRes := requestJSON(t, r, http.MethodGet, "/api/v1/vendor/shipments", nil, ownerLogin.AccessToken)
	if shipmentListRes.Code != http.StatusOK {
		t.Fatalf("list shipments status=%d body=%s", shipmentListRes.Code, shipmentListRes.Body.String())
	}

	var shipmentListPayload struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(shipmentListRes.Body.Bytes(), &shipmentListPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(shipmentListPayload.Items) != 1 {
		t.Fatalf("expected one shipment, got len=%d", len(shipmentListPayload.Items))
	}

	refundCreateRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/orders/"+orderPayload.Order.ID+"/refund-requests", map[string]interface{}{
		"shipment_id": shipmentListPayload.Items[0].ID,
		"reason":      "Changed mind",
	}, "", guestHeaders)
	if refundCreateRes.Code != http.StatusCreated {
		t.Fatalf("create refund request status=%d body=%s", refundCreateRes.Code, refundCreateRes.Body.String())
	}

	var refundPayload struct {
		RefundRequest struct {
			ID string `json:"id"`
		} `json:"refund_request"`
	}
	if err := json.Unmarshal(refundCreateRes.Body.Bytes(), &refundPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	approveRes := requestJSON(t, r, http.MethodPatch, "/api/v1/vendor/refund-requests/"+refundPayload.RefundRequest.ID+"/decision", map[string]string{
		"decision":        "approve",
		"decision_reason": "Approved for analytics flow",
	}, ownerLogin.AccessToken)
	if approveRes.Code != http.StatusOK {
		t.Fatalf("approve refund status=%d body=%s", approveRes.Code, approveRes.Body.String())
	}

	overviewRes := requestJSON(t, r, http.MethodGet, "/api/v1/vendor/analytics/overview", nil, ownerLogin.AccessToken)
	if overviewRes.Code != http.StatusOK {
		t.Fatalf("vendor analytics overview status=%d body=%s", overviewRes.Code, overviewRes.Body.String())
	}
	var overviewPayload struct {
		RevenueCents   int64 `json:"revenue_cents"`
		OrderCount     int   `json:"order_count"`
		PaidOrderCount int   `json:"paid_order_count"`
		ShipmentCount  int   `json:"shipment_count"`
		RefundStats    struct {
			ApprovedTotal int `json:"approved_total"`
		} `json:"refund_stats"`
	}
	if err := json.Unmarshal(overviewRes.Body.Bytes(), &overviewPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if overviewPayload.OrderCount != 1 || overviewPayload.PaidOrderCount != 1 || overviewPayload.ShipmentCount != 1 {
		t.Fatalf("unexpected overview counts %#v", overviewPayload)
	}
	if overviewPayload.RevenueCents <= 0 {
		t.Fatalf("expected positive revenue, got %d", overviewPayload.RevenueCents)
	}
	if overviewPayload.RefundStats.ApprovedTotal != 1 {
		t.Fatalf("expected approved refunds total 1, got %d", overviewPayload.RefundStats.ApprovedTotal)
	}

	topProductsRes := requestJSON(t, r, http.MethodGet, "/api/v1/vendor/analytics/top-products", nil, ownerLogin.AccessToken)
	if topProductsRes.Code != http.StatusOK {
		t.Fatalf("vendor analytics top products status=%d body=%s", topProductsRes.Code, topProductsRes.Body.String())
	}
	var topProductsPayload struct {
		Total int `json:"total"`
		Items []struct {
			ProductID string `json:"product_id"`
			UnitsSold int32  `json:"units_sold"`
		} `json:"items"`
	}
	if err := json.Unmarshal(topProductsRes.Body.Bytes(), &topProductsPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if topProductsPayload.Total < 1 || len(topProductsPayload.Items) < 1 {
		t.Fatalf("expected top products data, got total=%d len=%d", topProductsPayload.Total, len(topProductsPayload.Items))
	}
	if topProductsPayload.Items[0].ProductID != product.ID {
		t.Fatalf("expected top product id %s, got %s", product.ID, topProductsPayload.Items[0].ProductID)
	}
	if topProductsPayload.Items[0].UnitsSold < 1 {
		t.Fatalf("expected units sold to be positive, got %d", topProductsPayload.Items[0].UnitsSold)
	}

	couponAnalyticsRes := requestJSON(t, r, http.MethodGet, "/api/v1/vendor/analytics/coupons", nil, ownerLogin.AccessToken)
	if couponAnalyticsRes.Code != http.StatusOK {
		t.Fatalf("vendor analytics coupons status=%d body=%s", couponAnalyticsRes.Code, couponAnalyticsRes.Body.String())
	}
	var couponAnalyticsPayload struct {
		Total int `json:"total"`
		Items []struct {
			Code string `json:"code"`
		} `json:"items"`
	}
	if err := json.Unmarshal(couponAnalyticsRes.Body.Bytes(), &couponAnalyticsPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if couponAnalyticsPayload.Total != 1 || len(couponAnalyticsPayload.Items) != 1 {
		t.Fatalf("expected single coupon analytics record, got total=%d len=%d", couponAnalyticsPayload.Total, len(couponAnalyticsPayload.Items))
	}
	if couponAnalyticsPayload.Items[0].Code != "ANALYTICS10" {
		t.Fatalf("expected coupon code ANALYTICS10, got %s", couponAnalyticsPayload.Items[0].Code)
	}

	buyerForbidden := requestJSON(t, r, http.MethodGet, "/api/v1/vendor/analytics/overview", nil, buyer.AccessToken)
	if buyerForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected buyer forbidden for vendor analytics, got status=%d body=%s", buyerForbidden.Code, buyerForbidden.Body.String())
	}
}

func TestAdminAnalyticsOverviewRevenueAndVendors(t *testing.T) {
	cfg := testConfig()
	cfg.Environment = "development"
	cfg.DefaultCommission = 1000
	r := mustRouterWithConfig(t, cfg)

	finance := registerUser(t, r, "finance@example.com")
	support := registerUser(t, r, "support@example.com")
	buyer := registerUser(t, r, "buyer-admin-analytics@example.com")

	paidGuestToken := "gst_admin_analytics_paid"
	paidOrderID := createGuestOrderFromSeededCatalog(t, r, paidGuestToken, "idem-admin-analytics-paid-order")
	paidHeaders := map[string]string{guestTokenHeader: paidGuestToken}

	codRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/payments/cod/confirm", map[string]interface{}{
		"order_id":        paidOrderID,
		"idempotency_key": "idem-admin-analytics-cod-1",
	}, "", paidHeaders)
	if codRes.Code != http.StatusCreated {
		t.Fatalf("cod confirm status=%d body=%s", codRes.Code, codRes.Body.String())
	}

	paidOrderRes := requestJSONWithHeaders(t, r, http.MethodGet, "/api/v1/orders/"+paidOrderID, nil, "", paidHeaders)
	if paidOrderRes.Code != http.StatusOK {
		t.Fatalf("get paid order status=%d body=%s", paidOrderRes.Code, paidOrderRes.Body.String())
	}

	var paidOrderPayload struct {
		Order struct {
			Shipments []struct {
				ID string `json:"id"`
			} `json:"shipments"`
		} `json:"order"`
	}
	if err := json.Unmarshal(paidOrderRes.Body.Bytes(), &paidOrderPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(paidOrderPayload.Order.Shipments) == 0 {
		t.Fatal("expected at least one shipment for paid order")
	}

	refundCreateRes := requestJSONWithHeaders(t, r, http.MethodPost, "/api/v1/orders/"+paidOrderID+"/refund-requests", map[string]interface{}{
		"shipment_id": paidOrderPayload.Order.Shipments[0].ID,
		"reason":      "Admin analytics pending dispute fixture",
	}, "", paidHeaders)
	if refundCreateRes.Code != http.StatusCreated {
		t.Fatalf("create refund request status=%d body=%s", refundCreateRes.Code, refundCreateRes.Body.String())
	}

	_ = createGuestOrderFromSeededCatalog(t, r, "gst_admin_analytics_pending", "idem-admin-analytics-pending-order")

	overviewRes := requestJSON(t, r, http.MethodGet, "/api/v1/admin/dashboard/overview", nil, finance.AccessToken)
	if overviewRes.Code != http.StatusOK {
		t.Fatalf("admin overview status=%d body=%s", overviewRes.Code, overviewRes.Body.String())
	}
	var overviewPayload struct {
		PlatformRevenueCents  int64 `json:"platform_revenue_cents"`
		CommissionEarnedCents int64 `json:"commission_earned_cents"`
		OrderVolumes          struct {
			Total          int `json:"total"`
			PendingPayment int `json:"pending_payment"`
			CODConfirmed   int `json:"cod_confirmed"`
		} `json:"order_volumes"`
		VendorMetrics struct {
			TotalVendors int `json:"total_vendors"`
		} `json:"vendor_metrics"`
		ModerationQueue struct {
			PendingProducts int `json:"pending_products"`
		} `json:"moderation_queue"`
		Disputes struct {
			RefundRequestsTotal int `json:"refund_requests_total"`
			PendingTotal        int `json:"pending_total"`
		} `json:"disputes"`
	}
	if err := json.Unmarshal(overviewRes.Body.Bytes(), &overviewPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if overviewPayload.PlatformRevenueCents <= 0 {
		t.Fatalf("expected positive platform revenue, got %d", overviewPayload.PlatformRevenueCents)
	}
	if overviewPayload.CommissionEarnedCents <= 0 {
		t.Fatalf("expected positive commission earned, got %d", overviewPayload.CommissionEarnedCents)
	}
	if overviewPayload.OrderVolumes.Total != 2 {
		t.Fatalf("expected 2 total orders, got %d", overviewPayload.OrderVolumes.Total)
	}
	if overviewPayload.OrderVolumes.CODConfirmed != 1 {
		t.Fatalf("expected 1 cod_confirmed order, got %d", overviewPayload.OrderVolumes.CODConfirmed)
	}
	if overviewPayload.OrderVolumes.PendingPayment != 1 {
		t.Fatalf("expected 1 pending_payment order, got %d", overviewPayload.OrderVolumes.PendingPayment)
	}
	if overviewPayload.VendorMetrics.TotalVendors < 2 {
		t.Fatalf("expected at least two seeded vendors, got %d", overviewPayload.VendorMetrics.TotalVendors)
	}
	if overviewPayload.ModerationQueue.PendingProducts != 0 {
		t.Fatalf("expected no pending moderation products, got %d", overviewPayload.ModerationQueue.PendingProducts)
	}
	if overviewPayload.Disputes.RefundRequestsTotal != 1 || overviewPayload.Disputes.PendingTotal != 1 {
		t.Fatalf("expected one pending dispute, got %#v", overviewPayload.Disputes)
	}

	revenueRes := requestJSON(t, r, http.MethodGet, "/api/v1/admin/analytics/revenue?days=30", nil, finance.AccessToken)
	if revenueRes.Code != http.StatusOK {
		t.Fatalf("admin revenue analytics status=%d body=%s", revenueRes.Code, revenueRes.Body.String())
	}
	var revenuePayload struct {
		WindowDays int `json:"window_days"`
		Summary    struct {
			SettledOrdersTotal    int   `json:"settled_orders_total"`
			GrossRevenueCents     int64 `json:"gross_revenue_cents"`
			CommissionEarnedCents int64 `json:"commission_earned_cents"`
		} `json:"summary"`
		Points []struct {
			Date       string `json:"date"`
			OrderCount int    `json:"order_count"`
		} `json:"points"`
	}
	if err := json.Unmarshal(revenueRes.Body.Bytes(), &revenuePayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if revenuePayload.WindowDays != 30 {
		t.Fatalf("expected window_days=30, got %d", revenuePayload.WindowDays)
	}
	if revenuePayload.Summary.SettledOrdersTotal != 1 {
		t.Fatalf("expected one settled order, got %d", revenuePayload.Summary.SettledOrdersTotal)
	}
	if revenuePayload.Summary.GrossRevenueCents <= 0 || revenuePayload.Summary.CommissionEarnedCents <= 0 {
		t.Fatalf("expected positive revenue summary, got %#v", revenuePayload.Summary)
	}
	if len(revenuePayload.Points) != 30 {
		t.Fatalf("expected 30 daily revenue points, got %d", len(revenuePayload.Points))
	}

	vendorsRes := requestJSON(t, r, http.MethodGet, "/api/v1/admin/analytics/vendors", nil, finance.AccessToken)
	if vendorsRes.Code != http.StatusOK {
		t.Fatalf("admin vendors analytics status=%d body=%s", vendorsRes.Code, vendorsRes.Body.String())
	}
	var vendorsPayload struct {
		Total int `json:"total"`
		Items []struct {
			VendorID              string `json:"vendor_id"`
			SettledOrderCount     int    `json:"settled_order_count"`
			GrossRevenueCents     int64  `json:"gross_revenue_cents"`
			CommissionEarnedCents int64  `json:"commission_earned_cents"`
		} `json:"items"`
	}
	if err := json.Unmarshal(vendorsRes.Body.Bytes(), &vendorsPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if vendorsPayload.Total < 2 || len(vendorsPayload.Items) < 2 {
		t.Fatalf("expected at least two vendor analytics rows, got total=%d len=%d", vendorsPayload.Total, len(vendorsPayload.Items))
	}
	foundSettledVendor := false
	for _, item := range vendorsPayload.Items {
		if item.SettledOrderCount > 0 {
			if item.GrossRevenueCents <= 0 || item.CommissionEarnedCents <= 0 {
				t.Fatalf("expected settled vendor to have positive revenue + commission, got %#v", item)
			}
			foundSettledVendor = true
		}
	}
	if !foundSettledVendor {
		t.Fatal("expected at least one vendor with settled order analytics")
	}

	badDaysRes := requestJSON(t, r, http.MethodGet, "/api/v1/admin/analytics/revenue?days=0", nil, finance.AccessToken)
	if badDaysRes.Code != http.StatusBadRequest {
		t.Fatalf("expected bad days query to return 400, got status=%d body=%s", badDaysRes.Code, badDaysRes.Body.String())
	}

	supportForbidden := requestJSON(t, r, http.MethodGet, "/api/v1/admin/dashboard/overview", nil, support.AccessToken)
	if supportForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected support to be forbidden for admin analytics, got status=%d body=%s", supportForbidden.Code, supportForbidden.Body.String())
	}
	buyerForbidden := requestJSON(t, r, http.MethodGet, "/api/v1/admin/analytics/vendors", nil, buyer.AccessToken)
	if buyerForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected buyer to be forbidden for admin analytics, got status=%d body=%s", buyerForbidden.Code, buyerForbidden.Body.String())
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

func TestSecurityHeadersAndRequestSizeLimit(t *testing.T) {
	r := mustRouter(t)

	health := requestJSON(t, r, http.MethodGet, "/health", nil, "")
	if health.Code != http.StatusOK {
		t.Fatalf("health status=%d body=%s", health.Code, health.Body.String())
	}
	if got := health.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options nosniff, got %q", got)
	}
	if got := health.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("expected X-Frame-Options DENY, got %q", got)
	}
	if got := health.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("expected Referrer-Policy no-referrer, got %q", got)
	}

	sizeCfg := testConfig()
	sizeCfg.MaxRequestBodyBytes = 16
	sizeLimited := mustRouterWithConfig(t, sizeCfg)
	largeReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		strings.NewReader(`{"email":"large@example.com","password":"strong-password"}`),
	)
	largeReq.Header.Set("Content-Type", "application/json")

	largeRes := httptest.NewRecorder()
	sizeLimited.ServeHTTP(largeRes, largeReq)
	if largeRes.Code != http.StatusBadRequest {
		t.Fatalf("expected oversized body to be rejected, got status=%d body=%s", largeRes.Code, largeRes.Body.String())
	}
}

func TestAuthRateLimitingAppliesToAuthEndpoints(t *testing.T) {
	cfg := testConfig()
	cfg.EnableRateLimit = true
	cfg.GlobalRateLimitRPS = 1000
	cfg.GlobalRateLimitBurst = 1000
	cfg.AuthRateLimitRPS = 1
	cfg.AuthRateLimitBurst = 1

	r := mustRouterWithConfig(t, cfg)
	requestBody := map[string]string{
		"email":    "unknown@example.com",
		"password": "strong-password",
	}

	hitRateLimit := false
	for i := 0; i < 4; i++ {
		res := requestJSON(t, r, http.MethodPost, "/api/v1/auth/login", requestBody, "")
		if res.Code == http.StatusTooManyRequests {
			hitRateLimit = true
			break
		}
	}
	if !hitRateLimit {
		t.Fatal("expected auth rate limiter to return 429 under burst traffic")
	}
}

func TestCatalogAndAdminVendorPaginationValidation(t *testing.T) {
	cfg := testConfig()
	cfg.Environment = "development"
	r := mustRouterWithConfig(t, cfg)

	invalidCatalogLimit := requestJSON(t, r, http.MethodGet, "/api/v1/catalog/products?limit=0", nil, "")
	if invalidCatalogLimit.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid catalog limit to return 400, got status=%d body=%s", invalidCatalogLimit.Code, invalidCatalogLimit.Body.String())
	}
	invalidCatalogRange := requestJSON(t, r, http.MethodGet, "/api/v1/catalog/products?price_min=500&price_max=100", nil, "")
	if invalidCatalogRange.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid catalog price range to return 400, got status=%d body=%s", invalidCatalogRange.Code, invalidCatalogRange.Body.String())
	}

	validCatalog := requestJSON(t, r, http.MethodGet, "/api/v1/catalog/products?limit=1&offset=0", nil, "")
	if validCatalog.Code != http.StatusOK {
		t.Fatalf("expected valid catalog pagination status=200, got status=%d body=%s", validCatalog.Code, validCatalog.Body.String())
	}
	var catalogPayload struct {
		Items  []json.RawMessage `json:"items"`
		Total  int               `json:"total"`
		Limit  int               `json:"limit"`
		Offset int               `json:"offset"`
	}
	if err := json.Unmarshal(validCatalog.Body.Bytes(), &catalogPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if catalogPayload.Limit != 1 || catalogPayload.Offset != 0 {
		t.Fatalf("expected catalog pagination metadata limit=1 offset=0, got limit=%d offset=%d", catalogPayload.Limit, catalogPayload.Offset)
	}
	if len(catalogPayload.Items) > 1 {
		t.Fatalf("expected at most 1 catalog item, got %d", len(catalogPayload.Items))
	}

	admin := registerUser(t, r, "admin@example.com")
	ownerOne := registerUser(t, r, "vendor-owner-one@example.com")
	ownerTwo := registerUser(t, r, "vendor-owner-two@example.com")

	firstVendor := requestJSON(t, r, http.MethodPost, "/api/v1/vendors/register", map[string]string{
		"slug":         "hardening-vendor-one",
		"display_name": "Hardening Vendor One",
	}, ownerOne.AccessToken)
	if firstVendor.Code != http.StatusCreated {
		t.Fatalf("register vendor one status=%d body=%s", firstVendor.Code, firstVendor.Body.String())
	}
	secondVendor := requestJSON(t, r, http.MethodPost, "/api/v1/vendors/register", map[string]string{
		"slug":         "hardening-vendor-two",
		"display_name": "Hardening Vendor Two",
	}, ownerTwo.AccessToken)
	if secondVendor.Code != http.StatusCreated {
		t.Fatalf("register vendor two status=%d body=%s", secondVendor.Code, secondVendor.Body.String())
	}

	vendorList := requestJSON(t, r, http.MethodGet, "/api/v1/admin/vendors?limit=1&offset=1", nil, admin.AccessToken)
	if vendorList.Code != http.StatusOK {
		t.Fatalf("expected admin vendor list to return 200, got status=%d body=%s", vendorList.Code, vendorList.Body.String())
	}

	var vendorPayload struct {
		Items  []json.RawMessage `json:"items"`
		Total  int               `json:"total"`
		Limit  int               `json:"limit"`
		Offset int               `json:"offset"`
	}
	if err := json.Unmarshal(vendorList.Body.Bytes(), &vendorPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if vendorPayload.Total < 2 {
		t.Fatalf("expected total vendors >= 2, got %d", vendorPayload.Total)
	}
	if vendorPayload.Limit != 1 || vendorPayload.Offset != 1 {
		t.Fatalf("expected vendor pagination metadata limit=1 offset=1, got limit=%d offset=%d", vendorPayload.Limit, vendorPayload.Offset)
	}
	if len(vendorPayload.Items) != 1 {
		t.Fatalf("expected vendor list length 1, got %d", len(vendorPayload.Items))
	}

	invalidVendorLimit := requestJSON(t, r, http.MethodGet, "/api/v1/admin/vendors?limit=0", nil, admin.AccessToken)
	if invalidVendorLimit.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid vendor limit to return 400, got status=%d body=%s", invalidVendorLimit.Code, invalidVendorLimit.Body.String())
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
