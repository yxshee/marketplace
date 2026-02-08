package router

import (
	"net/http"

	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/auditlog"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/auth"
)

func actorTypeForRole(role auth.Role) string {
	switch role {
	case auth.RoleVendorOwner:
		return "vendor"
	case auth.RoleBuyer:
		return "buyer"
	default:
		return "admin"
	}
}

func (a *api) recordAuditLog(
	r *http.Request,
	action string,
	targetType string,
	targetID string,
	before interface{},
	after interface{},
	metadata interface{},
) {
	if a.auditLogs == nil {
		return
	}
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		return
	}

	_, _ = a.auditLogs.Record(auditlog.RecordInput{
		ActorType:  actorTypeForRole(identity.Role),
		ActorID:    identity.UserID,
		ActorRole:  identity.Role.String(),
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Before:     before,
		After:      after,
		Metadata:   metadata,
	})
}
