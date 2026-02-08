# Production Deployment (Vercel + Render)

This document is the production deployment reference for the monorepo:

- Frontend: `apps/web` (Next.js) deployed to Vercel
- Backend: `services/api` (Go) deployed to Render (Docker)

## Architecture

```text
Internet
  |
  |  https://<web-domain>               https://<api-domain>
  |  (Vercel)                           (Render Web Service)
  v
Next.js (apps/web)  ---- HTTPS ---->  Go API (services/api)
   |                                     |
   | server actions / route handlers      |
   v                                     v
httpOnly cookies (guest/admin/vendor)   Stripe Webhooks -> /api/v1/webhooks/stripe
```

Notes:
- The web app calls the API from the server (Vercel runtime) using `fetch`.
- Auth uses JWT access tokens passed to the API via `Authorization: Bearer ...` (not API cookies).
- Stripe webhook endpoint is implemented at `POST /api/v1/webhooks/stripe`.

## Repo/Build Facts (Audit)

- Package manager: `pnpm` (workspace)
- Web framework: Next.js (declared `^15.1.7`, resolved `15.5.12`) (React `^19`)
- API runtime: Go (CI uses Go `1.24`)
- API Dockerfile: `services/api/Dockerfile` (multi-stage + distroless)
- Health endpoints:
  - `GET /health`
  - `GET /healthz`
  - `GET /api/v1/health`
  - `GET /api/v1/healthz`

## Environment Variables

### Web (Vercel) (`apps/web`)

| Variable | Required | Example | Purpose |
|---|---:|---|---|
| `NEXT_PUBLIC_API_BASE_URL` | yes | `https://api.example.com/api/v1` | API base URL (safe for server + browser) |
| `MARKETPLACE_API_BASE_URL` | no | `https://api.example.com/api/v1` | Back-compat API base URL (server runtime fallback) |

Notes:
- Prefer `NEXT_PUBLIC_API_BASE_URL` for Vercel. The code still accepts `MARKETPLACE_API_BASE_URL` as a fallback.
- If you change the API domain, update this value and redeploy.

### API (Render) (`services/api`)

| Variable | Required | Example | Purpose |
|---|---:|---|---|
| `API_ENV` | yes | `production` | Enables production defaults (rate-limit enabled by default) |
| `API_PORT` | no | `8080` | Port to listen on (defaults to `8080`) |
| `API_CORS_ALLOW_ORIGINS` | no | `https://web.example.com,http://localhost:3000` | Allowed CORS origins (comma-separated) |
| `API_JWT_SECRET` | yes | `...` | JWT signing secret (access+refresh) |
| `API_JWT_ISSUER` | yes | `marketplace-api` | JWT issuer claim |
| `API_ACCESS_TOKEN_TTL_SECONDS` | no | `900` | Access token TTL |
| `API_REFRESH_TOKEN_TTL_SECONDS` | no | `1209600` | Refresh token TTL |
| `API_SUPER_ADMIN_EMAILS` | no | `admin@example.com` | Bootstrap RBAC role mapping |
| `API_SUPPORT_EMAILS` | no | `support@example.com` | Bootstrap RBAC role mapping |
| `API_FINANCE_EMAILS` | no | `finance@example.com` | Bootstrap RBAC role mapping |
| `API_CATALOG_MOD_EMAILS` | no | `mod@example.com` | Bootstrap RBAC role mapping |
| `API_DEFAULT_COMMISSION_BPS` | no | `1000` | Default commission in basis points |
| `API_STRIPE_MODE` | no | `live` | `mock` (default) or `live` (use Stripe API) |
| `API_STRIPE_SECRET_KEY` | if `API_STRIPE_MODE=live` | `sk_test_...` | Stripe secret key |
| `API_STRIPE_WEBHOOK_SECRET` | yes (for real Stripe webhooks) | `whsec_...` | Stripe webhook signature secret |
| `API_MAX_REQUEST_BODY_BYTES` | no | `1048576` | Request size limit |
| `API_RATE_LIMIT_ENABLED` | no | `true` | Enable global + auth rate limiting |
| `API_RATE_LIMIT_RPS` | no | `30` | Global rate limiter RPS |
| `API_RATE_LIMIT_BURST` | no | `90` | Global rate limiter burst |
| `API_AUTH_RATE_LIMIT_RPS` | no | `5` | Auth endpoints limiter RPS |
| `API_AUTH_RATE_LIMIT_BURST` | no | `10` | Auth endpoints limiter burst |

### Database / Redis / Storage (Current State)

This repository includes SQL migrations under `services/api/migrations` and `make migrate-up` expects `DATABASE_URL`,
but the current Go API implementation is in-memory (no Postgres/Redis clients in runtime code yet).

Provision these when the persistence layer is introduced:

| Variable | Required | Example | Purpose |
|---|---:|---|---|
| `DATABASE_URL` | not used by API runtime (today) | `postgres://...` | Used by migration tooling (`migrate`) |
| `REDIS_URL` | not used by API runtime (today) | `redis://...` | Future rate-limit/idempotency/session storage |
| `S3_*` | not used by API runtime (today) |  | Future product images / invoice storage |

## Deployment Checklist

### Step 0 (Audit PR)

- Confirm repo scripts + versions
- Enumerate environment variables
- Run local verification (lint/typecheck/tests/build) and capture outputs

### Step 1 (API: Render)

1. Create Render Web Service from `services/api/Dockerfile`.
2. Configure env vars (minimum: `API_ENV=production`, `API_JWT_SECRET`, `API_STRIPE_*` as needed).
3. Configure health check:
   - current: `GET /health`
4. Configure Stripe webhook in Stripe Dashboard:
   - endpoint: `https://<api-domain>/api/v1/webhooks/stripe`
   - secret: set as `API_STRIPE_WEBHOOK_SECRET`
5. Smoke-check:
   - `GET /health` => `200`
   - `GET /api/v1/catalog/products` => `200`
   - auth register/login works

### Step 2 (Web: Vercel)

1. Create Vercel project with Root Directory `apps/web`.
2. Set Vercel env var `NEXT_PUBLIC_API_BASE_URL=https://<api-domain>/api/v1`.
3. Deploy on merge to `main`.

### Step 3 (Prod Wiring)

- Set `WEB_BASE_URL` and `API_BASE_URL` in your operational docs (not in git).
- Add monitoring:
  - uptime check for `GET /health`

## Risks / Gotchas

- Persistence: current API state is in-memory (orders, auth sessions, etc.) and will reset on deploy/restart.
- CORS: API does not currently set CORS headers. If you later move API calls to the browser, add an explicit allowlist.
- Platform port: API uses `API_PORT` (default `8080`). Some hosts prefer `PORT`; keep this in mind when deploying.
- Stripe webhooks: must use the publicly reachable API domain and the correct `whsec_...` for that endpoint.

## Verification Outputs (Local)

Generated on 2026-02-09.

### Web (lint/typecheck/test/build)

Command:

```bash
pnpm -r lint && pnpm -r typecheck && pnpm -r test && pnpm -r build
```

Output:

```text
Scope: 2 of 3 workspace projects
packages/shared lint$ tsc --noEmit
packages/shared lint: Done
apps/web lint$ eslint .
apps/web lint: Done
Scope: 2 of 3 workspace projects
packages/shared typecheck$ tsc --noEmit
packages/shared typecheck: Done
apps/web typecheck$ tsc --noEmit
apps/web typecheck: Done
Scope: 2 of 3 workspace projects
packages/shared test$ vitest run --passWithNoTests
packages/shared test:  RUN  v3.2.4 /Users/venom/github/vendors/marketplace-gumroad-inspired/packages/shared
packages/shared test: No test files found, exiting with code 0
packages/shared test: include: **/*.{test,spec}.?(c|m)[jt]s?(x)
packages/shared test: exclude:  **/node_modules/**, **/dist/**, **/cypress/**, **/.{idea,git,cache,output,temp}/**, **/{karma,rollup,webpack,vite,vitest,jest,ava,babel,nyc,cypress,tsup,build,eslint,prettier}.config.*
packages/shared test: Done
apps/web test$ vitest run --passWithNoTests
apps/web test:  RUN  v3.2.4 /Users/venom/github/vendors/marketplace-gumroad-inspired/apps/web
apps/web test:  ✓ src/components/ui/surface-card.test.tsx (1 test) 12ms
apps/web test:  Test Files  1 passed (1)
apps/web test:       Tests  1 passed (1)
apps/web test:    Start at  01:51:16
apps/web test:    Duration  957ms (transform 44ms, setup 60ms, collect 106ms, tests 12ms, environment 331ms, prepare 63ms)
apps/web test: Done
Scope: 2 of 3 workspace projects
packages/shared build$ tsc -p tsconfig.json
packages/shared build: Done
apps/web build$ next build
apps/web build:    ▲ Next.js 15.5.12
apps/web build:    Creating an optimized production build ...
apps/web build:  ✓ Compiled successfully in 1859ms
apps/web build:    Linting and checking validity of types ...
apps/web build:    Collecting page data ...
apps/web build:    Generating static pages (0/9) ...
apps/web build:    Generating static pages (2/9) 
apps/web build:    Generating static pages (4/9) 
apps/web build:    Generating static pages (6/9) 
apps/web build:  ✓ Generating static pages (9/9)
apps/web build:    Finalizing page optimization ...
apps/web build:    Collecting build traces ...
apps/web build: Route (app)                                 Size  First Load JS
apps/web build: ┌ ƒ /                                      167 B         105 kB
apps/web build: ├ ○ /_not-found                            999 B         103 kB
apps/web build: ├ ƒ /admin                                 133 B         102 kB
apps/web build: ├ ƒ /api/invoices/[orderID]                133 B         102 kB
apps/web build: ├ ○ /buyer/search                          133 B         102 kB
apps/web build: ├ ƒ /cart                                  169 B         105 kB
apps/web build: ├ ƒ /categories/[slug]                     167 B         105 kB
apps/web build: ├ ƒ /checkout                              169 B         105 kB
apps/web build: ├ ƒ /checkout/confirmation                 169 B         105 kB
apps/web build: ├ ƒ /products/[productID]                  169 B         105 kB
apps/web build: ├ ƒ /search                                167 B         105 kB
apps/web build: └ ƒ /vendor                                133 B         102 kB
apps/web build: + First Load JS shared by all             102 kB
apps/web build:   ├ chunks/743-33720d133c383396.js       45.8 kB
apps/web build:   ├ chunks/8e6518bb-c26e82767f1faf66.js  54.2 kB
apps/web build:   └ other shared chunks (total)          1.89 kB
apps/web build: ○  (Static)   prerendered as static content
apps/web build: ƒ  (Dynamic)  server-rendered on demand
apps/web build: Done
```

### API (vet/test/build)

Command:

```bash
cd services/api && go vet ./... && go test ./... && go build ./...
```

Output:

```text
?   	github.com/yxshee/marketplace-platform/services/api/cmd/server	[no test files]
ok  	github.com/yxshee/marketplace-platform/services/api/internal/auditlog	(cached)
ok  	github.com/yxshee/marketplace-platform/services/api/internal/auth	(cached)
ok  	github.com/yxshee/marketplace-platform/services/api/internal/catalog	(cached)
ok  	github.com/yxshee/marketplace-platform/services/api/internal/commerce	(cached)
?   	github.com/yxshee/marketplace-platform/services/api/internal/config	[no test files]
ok  	github.com/yxshee/marketplace-platform/services/api/internal/coupons	(cached)
ok  	github.com/yxshee/marketplace-platform/services/api/internal/http/router	(cached)
ok  	github.com/yxshee/marketplace-platform/services/api/internal/invoices	(cached)
ok  	github.com/yxshee/marketplace-platform/services/api/internal/payments	(cached)
?   	github.com/yxshee/marketplace-platform/services/api/internal/platform/identifier	[no test files]
ok  	github.com/yxshee/marketplace-platform/services/api/internal/promotions	(cached)
ok  	github.com/yxshee/marketplace-platform/services/api/internal/refunds	(cached)
ok  	github.com/yxshee/marketplace-platform/services/api/internal/vendors	(cached)
```
