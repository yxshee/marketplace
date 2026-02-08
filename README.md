# marketplace-gumroad-inspired

Multi-vendor ecommerce marketplace monorepo with three surfaces:
- Buyer
- Vendor
- Admin

## Tech Stack
- Frontend: Next.js App Router + TypeScript + Tailwind CSS
- Backend: Go + Chi + PostgreSQL + Redis
- Shared contracts: TypeScript package with Zod schemas
- CI: GitHub Actions

## Repository Layout
- `apps/web` - buyer/vendor/admin frontend
- `services/api` - Go API service (source of truth for business logic)
- `packages/shared` - shared API contracts and schemas
- `docs` - architecture, API, design system, runbooks

## Getting Started

### Prerequisites
- Node 20+
- pnpm 9+
- Go 1.19+

### Install
```bash
pnpm install
```

### Run checks
```bash
pnpm -r lint
pnpm -r typecheck
pnpm -r test
pnpm -r build
cd services/api && go test ./...
```

### Run apps
```bash
# Web
cd apps/web && pnpm dev

# API
cd services/api && go run ./cmd/server
```

## Branch & PR Policy
- Use branch names:
  - `feat/<area>-<short-scope>`
  - `fix/<area>-<short-scope>`
  - `chore/<area>-<short-scope>`
  - `docs/<area>-<short-scope>`
- No direct commits to `main` after initialization.
- Every branch ends in a PR with verification evidence and screenshots for UI changes.
