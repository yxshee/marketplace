package auth

import "fmt"

// Role identifies a principal's permission context in the marketplace.
type Role string

const (
	RoleBuyer            Role = "buyer"
	RoleVendorOwner      Role = "vendor_owner"
	RoleSuperAdmin       Role = "super_admin"
	RoleSupport          Role = "support"
	RoleFinance          Role = "finance"
	RoleCatalogModerator Role = "catalog_moderator"
)

// Permission represents an action that can be authorized.
type Permission string

const (
	PermissionViewCatalog            Permission = "view_catalog"
	PermissionManageVendorProducts   Permission = "manage_vendor_products"
	PermissionModerateProducts       Permission = "moderate_products"
	PermissionManagePromotions       Permission = "manage_promotions"
	PermissionManageCommission       Permission = "manage_commission"
	PermissionManageVendorVerify     Permission = "manage_vendor_verification"
	PermissionViewAuditLogs          Permission = "view_audit_logs"
	PermissionManageShipmentStatuses Permission = "manage_shipment_statuses"
)

var permissionMatrix = map[Role]map[Permission]bool{
	RoleBuyer: {
		PermissionViewCatalog: true,
	},
	RoleVendorOwner: {
		PermissionViewCatalog:            true,
		PermissionManageVendorProducts:   true,
		PermissionManageShipmentStatuses: true,
	},
	RoleSupport: {
		PermissionViewCatalog:        true,
		PermissionManageVendorVerify: true,
		PermissionViewAuditLogs:      true,
	},
	RoleFinance: {
		PermissionViewCatalog:      true,
		PermissionManagePromotions: true,
		PermissionManageCommission: true,
		PermissionViewAuditLogs:    true,
	},
	RoleCatalogModerator: {
		PermissionViewCatalog:      true,
		PermissionModerateProducts: true,
		PermissionViewAuditLogs:    true,
	},
	RoleSuperAdmin: {
		PermissionViewCatalog:            true,
		PermissionManageVendorProducts:   true,
		PermissionModerateProducts:       true,
		PermissionManagePromotions:       true,
		PermissionManageCommission:       true,
		PermissionManageVendorVerify:     true,
		PermissionViewAuditLogs:          true,
		PermissionManageShipmentStatuses: true,
	},
}

func (r Role) String() string {
	return string(r)
}

func (p Permission) String() string {
	return string(p)
}

// IsAllowed checks a role/permission pair against the canonical RBAC matrix.
func IsAllowed(role Role, permission Permission) bool {
	rolePerms, exists := permissionMatrix[role]
	if !exists {
		return false
	}
	return rolePerms[permission]
}

// MustBeAllowed validates and returns an error useful for API handlers.
func MustBeAllowed(role Role, permission Permission) error {
	if IsAllowed(role, permission) {
		return nil
	}
	return fmt.Errorf("rbac forbidden: role=%s permission=%s", role, permission)
}
