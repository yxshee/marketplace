# Marketplace Platform - Architecture Plan

## 1. Product Scope Restatement

### Surfaces
- Buyer surface: discovery, cart, checkout, order tracking, wallet, reviews.
- Vendor surface: onboarding, product management, coupons, shipment operations, refund decisions, analytics.
- Admin surface: verification, moderation, order ops, promotions, audits, finance analytics.

### Core Marketplace Invariants
- Multi-vendor checkout creates one order with vendor-split shipments.
- Go API is the source of truth for commerce and authorization rules.
- Product visibility is gated by manual moderation.
- Vendor selling is gated by vendor verification.
- Stripe webhooks require signature validation and idempotent processing.
- COD flow is explicit and auditable.
- Invoice numbering is stable, unique, and auditable.

## 2. Final Architecture and Tech Decisions

### Monorepo
- `apps/web`: Next.js App Router frontend in TypeScript.
- `services/api`: Go API (Chi, pgx/sqlc, migrations, webhook handlers, PDF generation).
- `packages/shared`: shared TypeScript contracts and Zod schemas.
- `docs`: architecture, API specs, runbooks.

### Runtime
- Web app performs SSR/Server Components for catalog-centric pages.
- API owns all write flows and domain invariants.
- Postgres is the transactional system of record.
- Redis supports rate limiting, ephemeral cart/session helpers, idempotency caches.
- S3-compatible object storage uses presigned uploads from API.

### Delivery
- Web deploy target: Vercel.
- API deploy target: Render.
- CI target: GitHub Actions, including DB/Redis service containers.

## 3. Security and Authorization Model

### Authentication
- First-party auth in Go API.
- Access tokens + refresh sessions with secure cookie handling.
- Session rotation and revocation supported.

### Authorization
- RBAC enforced at API middleware and handler service layer.
- Admin roles: `super_admin`, `support`, `finance`, `catalog_moderator`.
- Vendor role: single owner per vendor account.

### Security Baseline
- Zod validation on web inputs and Go-side validation in API request DTOs.
- SQL parameterization through sqlc queries only.
- Stripe webhook signature verification with replay/idempotency protection.
- Audit logs for all sensitive admin actions and key vendor actions.

## 4. Milestone Plan and Acceptance Criteria

| Step | Branch | Scope | Exit Criteria |
|---|---|---|---|
| 0 | `main` | Repo initialization | Private repo + scaffold + tag `v0.0.0` |
| 1 | `docs/architecture-plan` | Architecture and milestone docs | Plan approved and merged |
| 2 | `chore/monorepo-tooling` | Tooling, CI, baseline app/api build | Lint/typecheck/test/build pass |
| 3 | `feat/api-foundation` | DB schema, migrations, auth, RBAC skeleton | Auth and RBAC tests pass |
| 4a-4e | `feat/buyer-*` | Buyer catalog/checkout/payment/COD/invoices | Flow tests pass and screenshots included |
| 5a-5e | `feat/vendor-*` | Vendor onboarding/ops/refunds/analytics | Vendor-scoped authorization validated |
| 6a-6f | `feat/admin-*` | Admin ops/moderation/verification/analytics | Admin role matrix tests pass |
| 7 | `fix/security-hardening` | Hardening and performance baseline | Security checklist complete |
| 8 | `chore/final-polish` | Cleanup and release docs | CI green, docs complete, `v1.0.0` tagged |

## 5. Key Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Scope exceeds single delivery cycle | Delays and unstable quality | Strict branch slicing and PR size limits |
| Payment/webhook edge cases | Financial inconsistency | Event ledger + idempotency keys + deterministic state transitions |
| RBAC drift between UI/API | Privilege escalation bugs | API-first permission checks + table-driven tests |
| UI inconsistency across surfaces | Brand and UX degradation | Shared design tokens + UI primitives and spacing scale |
| Local infra mismatch | Incomplete verification | CI service containers for Postgres/Redis |

## 6. Acceptance Checklist for Every PR

- Conventional commit messages.
- Branch naming policy followed.
- Lint/typecheck/tests/build run and outputs captured.
- Screenshots attached for UI changes.
- No dead code, no placeholder TODOs in critical flow paths.
- Security checks done for auth, validation, payments, webhooks, upload paths.

## 7. Deferred Decisions Explicitly Out of V1

- Automated vendor payout disbursements.
- Product variant-level inventory modeling.
- Multi-currency checkout and invoicing.
- Full global shipping-rate engine.
