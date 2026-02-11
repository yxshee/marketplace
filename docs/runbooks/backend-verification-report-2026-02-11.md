# Backend Verification Report (2026-02-11)

## Scope and evidence
- Branch: `chore_backend_verification`
- Baseline note: `docs/runbooks/backend-verification-baseline-2026-02-11.md`
- Final verification commands run:
  - `pnpm -r run lint`
  - `pnpm -r run typecheck`
  - `pnpm -r run test`
  - `pnpm -r run build`
  - `cd services/api && go test ./...`
  - `cd services/api && make test`
  - `cd services/api && make lint`
  - `cd services/api && make sqlc-generate`
- Result: all commands above passed in this branch.

## Routes verified
- Router surface extracted from `services/api/internal/http/router/router.go`: 61 method+path routes.
- OpenAPI surface extracted from `services/api/openapi/openapi.yaml`: 61 method+path routes.
- Route parity check result:
  - router-only routes: 0
  - openapi-only routes: 0
- Fix applied: documented `/healthz` alias in OpenAPI and endpoint docs.

## RBAC coverage summary
- Existing RBAC handler-level tests were kept and re-verified.
- Added matrix-complete RBAC assertion in `services/api/internal/auth/rbac_test.go`:
  - validates every known permission for each role (`buyer`, `vendor_owner`, `support`, `finance`, `catalog_moderator`, `super_admin`).
  - prevents drift between role matrix and permission checks.

## Idempotency coverage summary
- Re-verified existing idempotency tests in commerce/payment/router layers.
- Added idempotency scope tests:
  - `services/api/internal/commerce/service_test.go`
    - actor-scoped idempotency key behavior
    - replay safety when same key is retried after cart changes
  - `services/api/internal/payments/service_test.go`
    - same idempotency key used across different orders remains order-scoped and safe

## Checkout split invariant tests summary
- Existing multi-vendor split tests re-verified.
- Added single-vendor explicit invariant test in `services/api/internal/commerce/service_test.go`:
  - one order with one vendor creates exactly one shipment
  - shipment totals are consistent with order totals
- Added edge-case coverage in `services/api/internal/commerce/service_test.go`:
  - empty cart quote/place-order
  - invalid SKU/product snapshot
  - zero/negative quantity
  - insufficient stock
  - invalid actor context

## Webhook verification summary
- Existing Stripe webhook tests re-verified:
  - valid signature accepted
  - invalid signature rejected
  - duplicate event deduplicated
  - concurrent duplicate deliveries deduplicated
  - failed order-sync does not mark event as permanently processed
- Verified no secret-bearing values were introduced in logs/tests.

## Presigned upload verification summary
- No presigned-upload API endpoints are currently registered in:
  - `services/api/internal/http/router/router.go`
  - `services/api/openapi/openapi.yaml`
- Result: unable to execute endpoint-level auth/key/TTL tests for presigned uploads in this branch.
- Status: documented as a known gap for follow-up implementation/testing.

## Seed behavior summary
- Added and passed `TestSeedCatalogRunsOnlyInDevelopment` in `services/api/internal/http/router/router_test.go`:
  - non-development (`production`) startup does not seed catalog
  - development startup seeds catalog as expected
- Existing seeded checkout/payment integration tests remain passing.

## Shared contracts and client validation summary
- Added runtime schema tests in `packages/shared/src/schemas/common.test.ts`.
- Aligned shared error contract with API error envelope:
  - `packages/shared/src/contracts/api.ts` now reflects `{"error":"<message>"}`.
- Added critical web API client request-shape tests in `apps/web/src/lib/api-client.test.ts`.

## Fixes and commit links
- API/backend invariant and route parity hardening:
  - [ca524ae](https://github.com/yxshee/marketplace-gumroad-inspired/commit/ca524ae)
- Shared/web contract and request-shape validation:
  - [f6a0dde](https://github.com/yxshee/marketplace-gumroad-inspired/commit/f6a0dde)

## Known gaps and next steps
- Presigned S3-compatible upload endpoints are not yet present in API/OpenAPI, so presign auth/key/size/TTL verification could not be completed.
- If presigned uploads are required for current release scope, add explicit endpoints and tests in a focused follow-up PR:
  - auth required
  - object key normalization and traversal protection
  - content-type/size constraints
  - short-lived presign TTL
