package router

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/auditlog"
)

type adminAuditLogListResponse struct {
	Items []auditlog.Entry `json:"items"`
	Total int              `json:"total"`
}

func (a *api) handleAdminAuditLogsList(w http.ResponseWriter, r *http.Request) {
	limit := 50
	limitRaw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if limitRaw != "" {
		parsed, err := strconv.Atoi(limitRaw)
		if err != nil || parsed < 1 || parsed > 200 {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 200")
			return
		}
		limit = parsed
	}

	offset := 0
	offsetRaw := strings.TrimSpace(r.URL.Query().Get("offset"))
	if offsetRaw != "" {
		parsed, err := strconv.Atoi(offsetRaw)
		if err != nil || parsed < 0 {
			writeError(w, http.StatusBadRequest, "offset must be zero or positive")
			return
		}
		offset = parsed
	}

	result := a.auditLogs.List(auditlog.ListInput{
		ActorType:  strings.TrimSpace(r.URL.Query().Get("actor_type")),
		ActorID:    strings.TrimSpace(r.URL.Query().Get("actor_id")),
		Action:     strings.TrimSpace(r.URL.Query().Get("action")),
		TargetType: strings.TrimSpace(r.URL.Query().Get("target_type")),
		TargetID:   strings.TrimSpace(r.URL.Query().Get("target_id")),
		Limit:      limit,
		Offset:     offset,
	})

	writeJSON(w, http.StatusOK, adminAuditLogListResponse{
		Items: result.Items,
		Total: result.Total,
	})
}
