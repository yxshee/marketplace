# System Overview

## Runtime topology
- Web (`apps/web`): Next.js App Router app for buyer, vendor, and admin surfaces.
- API (`services/api`): Go + Chi REST API that enforces business invariants.
- Shared contracts (`packages/shared`): TypeScript endpoint contracts and Zod schemas.
- Data stores:
  - PostgreSQL for transactional data
  - Redis for ephemeral controls (rate limiting/idempotency/cache patterns)
  - S3-compatible object storage for product media uploads (presigned flow)

## Core commerce invariants
- One checkout can contain products from multiple vendors.
- Checkout creates:
  - one order,
  - one shipment per vendor,
  - order/shipment financial snapshots.
- Payments are order-level, fulfillment/refunds are shipment-scoped.
- Products are buyer-visible only after moderation approval.
- Vendor selling requires verified vendor state.

## Security model
- Authentication: first-party auth with access/refresh tokens.
- Authorization: API-side RBAC matrix with role permission checks.
- Payment hardening:
  - idempotency keys for create/confirm operations,
  - Stripe signature verification,
  - webhook event deduplication.
- API hardening:
  - request body size limits,
  - secure response headers,
  - IP rate limiting (global + auth-specific),
  - validated pagination and query bounds on list/search endpoints.

## Surface responsibilities
### Buyer
- Catalog search, cart, checkout, payment confirmation, order and invoice retrieval.

### Vendor
- Onboarding, product/coupon management, shipment workflow, refund decisions, analytics.

### Admin
- Vendor verification, product moderation, order ops, promotions, audit logs, analytics, payment settings.

## Deployment model
- Web deploy target: Vercel.
- API deploy target: Render.
- CI pipeline: GitHub Actions with separate web and API jobs.
