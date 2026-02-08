# RBAC Matrix (V1)

## Roles
- Buyer (guest or authenticated)
- Vendor Owner
- Admin: `super_admin`, `support`, `finance`, `catalog_moderator`

## Capability Matrix

| Capability | Buyer | Vendor Owner | Support | Finance | Catalog Moderator | Super Admin |
|---|---:|---:|---:|---:|---:|---:|
| View public catalog | Yes | Yes | Yes | Yes | Yes | Yes |
| Manage own vendor products | No | Yes | No | No | No | Yes |
| Submit product for moderation | No | Yes | No | No | No | Yes |
| Approve/reject products | No | No | No | No | Yes | Yes |
| Manage vendor coupons | No | Yes | No | No | No | Yes |
| Manage platform promotions | No | No | No | Yes | No | Yes |
| View all orders | Own only | Own shipments | Yes | Yes | Limited | Yes |
| Update shipment statuses | No | Own shipments | No | No | No | Yes |
| Decide refunds for own shipments | No | Yes | No | No | No | Yes |
| Handle disputes/case operations | No | No | Yes | Yes | No | Yes |
| Manage commission settings | No | No | No | Yes | No | Yes |
| Manage vendor verification | No | No | Yes | No | No | Yes |
| Access audit logs | No | Own action log only | Yes | Yes | Yes | Yes |

## Enforcement Notes
- All checks are enforced in API middleware and handler-level ownership checks.
- UI checks are convenience only and never treated as security controls.
- Ownership checks compare authenticated principal to resource owner/vendor owner relation.
