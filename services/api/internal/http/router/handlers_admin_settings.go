package router

import (
	"net/http"

	"github.com/yxshee/marketplace-platform/services/api/internal/payments"
)

type paymentSettingsResponse struct {
	payments.PaymentSettings
}

type paymentSettingsPatchRequest struct {
	StripeEnabled *bool `json:"stripe_enabled"`
	CODEnabled    *bool `json:"cod_enabled"`
}

func (a *api) handleAdminPaymentSettingsGet(w http.ResponseWriter, _ *http.Request) {
	settings := a.payments.GetSettings()
	writeJSON(w, http.StatusOK, paymentSettingsResponse{PaymentSettings: settings})
}

func (a *api) handleAdminPaymentSettingsPatch(w http.ResponseWriter, r *http.Request) {
	var req paymentSettingsPatchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.StripeEnabled == nil && req.CODEnabled == nil {
		writeError(w, http.StatusBadRequest, "at least one settings field is required")
		return
	}

	previous := a.payments.GetSettings()
	settings := a.payments.UpdateSettings(payments.PaymentSettingsUpdate{
		StripeEnabled: req.StripeEnabled,
		CODEnabled:    req.CODEnabled,
	})
	a.recordAuditLog(
		r,
		"payment_settings_updated",
		"payment_settings",
		"default",
		previous,
		settings,
		nil,
	)

	writeJSON(w, http.StatusOK, paymentSettingsResponse{PaymentSettings: settings})
}
