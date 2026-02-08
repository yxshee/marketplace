# Seed Data Runbook

## Activation
Seed data is loaded automatically when API starts with:
- `API_ENV=development`

Implementation entrypoint:
- `services/api/internal/http/router/seed_catalog.go`

## Seeded entities
### Vendors (verified)
- `north-studio` (`North Studio`)
- `line-press` (`Line Press`)

### Categories
- `stationery`
- `prints`
- `home`

### Products
- Grid Notebook
- Desk Weekly Planner
- Monochrome Poster Print
- Ceramic Coffee Cup

## Behavior notes
- Seeding is idempotent and only runs when no visible products exist.
- Seed data is intended for local/dev previews and integration tests, not production.
