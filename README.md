# marketplace-gumroad-inspired

Production-grade multi-vendor ecommerce marketplace with three surfaces:
- Buyer
- Vendor
- Admin

UI/UX direction follows Gumroad-inspired principles (minimal, whitespace-first, crisp typography) with original components and assets.

## Current release
- Target release: `v1.0.0`
- Web: Next.js App Router + TypeScript
- API: Go + Chi (business-rule source of truth)

## Core capabilities
- Multi-vendor shared catalog with moderation gating
- Multi-shipment checkout (single order split by vendor)
- Stripe + COD payment flows with idempotency and webhook verification
- Vendor verification lifecycle and admin role-based operations
- Refund request + vendor decision flows
- Invoice PDF generation and retrieval
- Vendor/admin analytics and audit logging

## Repository layout
- `apps/web` - Buyer/Vendor/Admin frontend
- `services/api` - Go API service
- `packages/shared` - Shared TS contracts + Zod schemas
- `docs` - Architecture, API docs, runbooks, tracking, screenshots

## Prerequisites
- Node.js 22+
- pnpm 10+
- Go 1.24+

## Local setup
```bash
pnpm install
```

### Run API
```bash
cd services/api
go run ./cmd/server
```

### Run web
```bash
cd apps/web
pnpm dev
```

Default ports:
- API: `http://localhost:8080`
- Web: `http://localhost:3000`

## Environment variables
### API
- `API_ENV` (`development`, `test`, `production`)
- `API_PORT` (default `8080`)
- `API_JWT_SECRET`
- `API_JWT_ISSUER`
- `API_SUPER_ADMIN_EMAILS`
- `API_SUPPORT_EMAILS`
- `API_FINANCE_EMAILS`
- `API_CATALOG_MOD_EMAILS`
- `API_DEFAULT_COMMISSION_BPS`
- `API_STRIPE_MODE` (`mock`, `live`)
- `API_STRIPE_SECRET_KEY` (required for live mode)
- `API_STRIPE_WEBHOOK_SECRET`
- `API_MAX_REQUEST_BODY_BYTES`
- `API_RATE_LIMIT_ENABLED`
- `API_RATE_LIMIT_RPS`
- `API_RATE_LIMIT_BURST`
- `API_AUTH_RATE_LIMIT_RPS`
- `API_AUTH_RATE_LIMIT_BURST`

### Web
- `MARKETPLACE_API_BASE_URL` (default `http://localhost:8080/api/v1`)

## Quality gates
Run before pushing:
```bash
pnpm -r lint
pnpm -r typecheck
pnpm -r test
pnpm -r build
cd services/api && go test ./...
```

## Development seed data
When API runs with `API_ENV=development`, the service seeds:
- Verified vendors: `north-studio`, `line-press`
- Categories: `stationery`, `prints`, `home`
- Buyer-visible sample products

See `services/api/internal/http/router/seed_catalog.go` and `docs/runbooks/seed-data.md`.

## CI/CD
- GitHub Actions CI:
  - Web lint/typecheck/test/build
  - API vet/test/lint
- Web deploy target: Vercel
- API deploy target: Render
- API container: `services/api/Dockerfile`

## Branch and PR policy
- Branch names:
  - `feat/<area>-<short-scope>`
  - `fix/<area>-<short-scope>`
  - `chore/<area>-<short-scope>`
  - `docs/<area>-<short-scope>`
- No direct commits to `main` after initialization
- Every branch merges via PR with:
  - Scope summary
  - Verification checklist
  - Command outputs
  - Screenshots for UI changes

## Documentation index
- Docs index: `docs/README.md`
- Architecture: `docs/architecture/README.md`
- API: `docs/api/README.md`
- Runbooks: `docs/runbooks/README.md`
- Design system: `docs/design-system.md`
- Milestone tracking: `docs/tracking/`
