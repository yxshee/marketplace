# marketplace-gumroad-inspired

Multi-vendor ecommerce marketplace monorepo.

## Surfaces
- Buyer
- Vendor
- Admin

## Monorepo Layout (planned)
- `apps/web` - Next.js frontend
- `services/api` - Go API
- `packages/shared` - shared TS contracts/schemas
- `docs` - architecture, API, and runbooks

## Workflow
- `main` is protected and receives PR merges only after initialization.
- Branch naming:
  - `feat/<area>-<short-scope>`
  - `fix/<area>-<short-scope>`
  - `chore/<area>-<short-scope>`
  - `docs/<area>-<short-scope>`

See `docs/README.md` for documentation structure.
