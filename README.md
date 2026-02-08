<div align="center">
  <h1>marketplace-gumroad-inspired</h1>
  <p><strong>Production-grade multi-vendor ecommerce marketplace</strong></p>
  <p>Buyer, Vendor, and Admin surfaces with a clean, whitespace-first, Gumroad-inspired UI philosophy.</p>
</div>

<p align="center">
  <a href="https://github.com/yxshee/marketplace-gumroad-inspired/actions/workflows/ci.yml">
    <img alt="CI" src="https://img.shields.io/github/actions/workflow/status/yxshee/marketplace-gumroad-inspired/ci.yml?branch=main&label=CI" />
  </a>
  <a href="https://github.com/yxshee/marketplace-gumroad-inspired/releases/tag/v1.0.0">
    <img alt="Release" src="https://img.shields.io/github/v/tag/yxshee/marketplace-gumroad-inspired?label=release" />
  </a>
  <img alt="Go" src="https://img.shields.io/badge/go-1.24-00ADD8" />
  <img alt="Next.js" src="https://img.shields.io/badge/next.js-15-black" />
  <img alt="TypeScript" src="https://img.shields.io/badge/typescript-strict-3178C6" />
  <a href="./LICENSE">
    <img alt="License" src="https://img.shields.io/github/license/yxshee/marketplace-gumroad-inspired" />
  </a>
</p>

## Why This Project Exists

This repository delivers a marketplace architecture where the **Go API is the source of truth** for business invariants while Next.js provides fast, minimal, and consistent UX across all surfaces.

Key principles:
- Minimal UI, original components, consistent spacing/typography.
- Clear separation of concerns between frontend, API, and shared contracts.
- Security-first flows for auth, RBAC, payments, uploads, and auditability.

## Product Surfaces

| Surface | Primary Users | Core Outcomes |
| --- | --- | --- |
| Buyer | Guests + logged-in customers | Discovery, checkout, orders, invoices, reviews, wallet |
| Vendor | Vendor owner | Product lifecycle, coupons, shipment ops, refund decisions, analytics |
| Admin | Super admin, support, finance, catalog moderator | Verification, moderation, promotions, operations, audit + platform analytics |

## System Snapshot

| Layer | Stack | Responsibility |
| --- | --- | --- |
| Web | Next.js App Router + TypeScript + Tailwind | Buyer/Vendor/Admin UI, SSR pages, accessible interactions |
| API | Go + Chi + PostgreSQL + Redis | Domain logic, RBAC, checkout splitting, payments, moderation, invoicing |
| Contracts | `packages/shared` (TypeScript + Zod) | Shared API contracts and schema validation |
| Infrastructure | GitHub Actions + Vercel + Render + Docker | CI, deploy, and runtime parity |

## Architecture At A Glance

```mermaid
flowchart LR
  B[Buyer UI] --> W[Next.js Web App]
  V[Vendor UI] --> W
  A[Admin UI] --> W
  W -->|REST /api/v1| API[Go API (Chi)]

  API --> PG[(PostgreSQL)]
  API --> R[(Redis)]
  API --> S3[(S3 Compatible Storage)]
  API --> ST[Stripe]

  ST -->|Webhook| API
  API --> INV[PDF Invoices]
  API --> EVT[Audit + Event Logs]
```

## Core Capabilities

| Commerce | Governance | Operations |
| --- | --- | --- |
| Multi-vendor shared catalog | Product moderation workflow | Stripe + COD payment flows |
| Multi-shipment checkout per order | Vendor verification lifecycle | Idempotent webhook processing |
| Coupon and promotion model | Role-based admin controls | Invoice generation and download |
| Search and discovery filters | Audit logging for critical actions | Vendor and platform analytics |

## Repository Layout

```text
.
├── apps/web                  # Next.js frontend (buyer/vendor/admin)
├── services/api              # Go API (domain source of truth)
├── packages/shared           # Shared TS contracts and Zod schemas
├── docs
│   ├── architecture          # System-level architecture docs
│   ├── api                   # Endpoint and API reference docs
│   ├── runbooks              # Deployment/release/seed runbooks
│   └── tracking              # Milestone tracking artifacts
└── .github/workflows         # CI pipelines
```

## Local Development

### Prerequisites

- Node.js 22+
- pnpm 10+
- Go 1.24+

### 1) Install dependencies

```bash
pnpm install
```

### 2) Start API

```bash
cd services/api
go run ./cmd/server
```

### 3) Start web app

```bash
cd apps/web
pnpm dev
```

Default ports:
- API: `http://localhost:8080`
- Web: `http://localhost:3000`

## Environment Variables

<details>
<summary><strong>API variables</strong></summary>

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

</details>

<details>
<summary><strong>Web variables</strong></summary>

- `MARKETPLACE_API_BASE_URL` (default `http://localhost:8080/api/v1`)

</details>

## Quality Gates

Run before pushing:

```bash
pnpm -r lint
pnpm -r typecheck
pnpm -r test
pnpm -r build
cd services/api && go test ./...
```

## Seed Data

When API runs with `API_ENV=development`, the service seeds:
- Verified vendors: `north-studio`, `line-press`
- Categories: `stationery`, `prints`, `home`
- Buyer-visible sample products

Reference:
- `services/api/internal/http/router/seed_catalog.go`
- `docs/runbooks/seed-data.md`

## CI/CD And Deployment Targets

- CI workflow: `.github/workflows/ci.yml`
- Web deployment target: Vercel
- API deployment target: Render
- API container image build: `services/api/Dockerfile`

## Branching And PR Rules

- Branch naming patterns:
  - `feat/<area>-<short-scope>`
  - `fix/<area>-<short-scope>`
  - `chore/<area>-<short-scope>`
  - `docs/<area>-<short-scope>`
- No direct commits to `main` after initialization.
- Every branch merges through PR with:
  - Scope summary
  - Verification checklist
  - Command outputs
  - Screenshots for UI changes

## Documentation Hub

- Docs index: `docs/README.md`
- Architecture: `docs/architecture/README.md`
- API docs: `docs/api/README.md`
- Runbooks: `docs/runbooks/README.md`
- Design system: `docs/design-system.md`
- Milestone tracking: `docs/tracking/`
