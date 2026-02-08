# fix/security-hardening

Status: Ready for review.

## Implemented scope
- Added API hardening middleware:
  - configurable request body size limits,
  - secure default response headers,
  - configurable IP-based rate limiting (global + stricter auth endpoints).
- Hardened runtime server settings with explicit read/write/header/idle timeouts.
- Added strict query validation for catalog filters (`limit`, `offset`, `price_min`, `price_max`, `min_rating`).
- Added consistent pagination support (`limit`, `offset`) on key list endpoints:
  - vendor products,
  - vendor shipments,
  - vendor refund requests,
  - admin vendors,
  - admin moderation queue,
  - admin orders,
  - admin promotions,
  - admin vendor analytics.
- Extended shared frontend/backend contracts for paginated list metadata (`limit`, `offset`).
- Updated OpenAPI docs for the new pagination/query constraints and response metadata.
- Added router-level tests for:
  - security headers and oversized body rejection,
  - auth endpoint rate limiting,
  - catalog validation rules and admin vendor pagination bounds.

## Completion checklist
- [x] Implementation complete
- [x] pnpm -r lint
- [x] pnpm -r typecheck
- [x] pnpm -r test
- [x] pnpm -r build
- [x] go test ./... (API)
- [x] UI screenshots included (if UI changes)
- [x] Proof outputs added to PR description
