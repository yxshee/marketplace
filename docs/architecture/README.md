# Architecture

## Documents
- `architecture-plan.md` - Requirements, final tech choices, milestones, risks
- `system-overview.md` - Runtime topology, data flow, and core invariants
- `rbac-matrix.md` - Permission matrix for buyer/vendor/admin roles

## Principles
- Go API is the source of truth for commerce/business rules
- RBAC is enforced in API middleware and handler logic
- UI follows minimal, high-legibility, low-clutter design system
- Security defaults: validation, idempotency, auditable actions, webhook verification
