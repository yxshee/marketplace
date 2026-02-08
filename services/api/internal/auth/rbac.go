package auth

import (
	"fmt"
	"sort"
)

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
	PermissionViewCatalog              Permission = "view_catalog"
	PermissionManageVendorProducts     Permission = "manage_vendor_products"
	PermissionManageVendorCoupons      Permission = "manage_vendor_coupons"
	PermissionManageShipmentOrders     Permission = "manage_shipment_orders"
	PermissionManageRefundDecisions    Permission = "manage_refund_decisions"
	PermissionViewVendorAnalytics      Permission = "view_vendor_analytics"
	PermissionManageVendorVerification Permission = "manage_vendor_verification"
	PermissionModerateProducts         Permission = "moderate_products"
	PermissionManageOrdersOperations   Permission = "manage_orders_operations"
	PermissionManagePromotions         Permission = "manage_promotions"
	PermissionManageCommission         Permission = "manage_commission"
	PermissionManagePaymentSettings    Permission = "manage_payment_settings"
	PermissionManageTaxSettings        Permission = "manage_tax_settings"
	PermissionViewAdminAnalytics       Permission = "view_admin_analytics"
	PermissionViewAuditLogs            Permission = "view_audit_logs"
)

var permissionMatrix = map[Role]map[Permission]bool{
	RoleBuyer: {
		PermissionViewCatalog: true,
	},
	RoleVendorOwner: {
		PermissionViewCatalog:           true,
		PermissionManageVendorProducts:  true,
		PermissionManageVendorCoupons:   true,
		PermissionManageShipmentOrders:  true,
		PermissionManageRefundDecisions: true,
		PermissionViewVendorAnalytics:   true,
	},
	RoleSupport: {
		PermissionViewCatalog:              true,
		PermissionManageOrdersOperations:   true,
		PermissionManageVendorVerification: true,
		PermissionViewAuditLogs:            true,
	},
	RoleFinance: {
		PermissionViewCatalog:           true,
		PermissionManagePromotions:      true,
		PermissionManageCommission:      true,
		PermissionManagePaymentSettings: true,
		PermissionManageTaxSettings:     true,
		PermissionViewAdminAnalytics:    true,
		PermissionViewAuditLogs:         true,
	},
	RoleCatalogModerator: {
		PermissionViewCatalog:      true,
		PermissionModerateProducts: true,
		PermissionViewAuditLogs:    true,
	},
	RoleSuperAdmin: {},
}

func init() {
	for permission := range allPermissionsSet() {
		permissionMatrix[RoleSuperAdmin][permission] = true
	}
}

func allPermissionsSet() map[Permission]struct{} {
	all := make(map[Permission]struct{})
	for _, rolePerms := range permissionMatrix {
		for permission := range rolePerms {
			all[permission] = struct{}{}
		}
	}
	return all
}

// Roles returns the list of known roles in stable order.
func Roles() []Role {
	roles := []Role{
		RoleBuyer,
		RoleVendorOwner,
		RoleSupport,
		RoleFinance,
		RoleCatalogModerator,
		RoleSuperAdmin,
	}
	return roles
}

// Permissions returns all known permissions in stable order.
func Permissions() []Permission {
	all := make([]Permission, 0, len(allPermissionsSet()))
	for permission := range allPermissionsSet() {
		all = append(all, permission)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i] < all[j]
	})
	return all
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
