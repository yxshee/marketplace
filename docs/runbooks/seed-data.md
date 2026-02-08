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
- `koi-workshop` (`Koi Workshop`)
- `sunbeam-supply` (`Sunbeam Supply`)
- `fern-atelier` (`Fern Atelier`)
- `moss-and-mortar` (`Moss & Mortar`)

### Categories
- `stationery`
- `prints`
- `home`
- `desk`
- `kitchen`
- `accessories`
- `apparel`
- `outdoors`

### Products
The development seed includes 30 products across the vendors above.

Example products:
- Grid Notebook
- Risograph Print: Blue Hour
- Hardwood Desk Tray
- Glass Spice Jar Set
- Relaxed Crew Tee
- Wool Picnic Blanket

## Behavior notes
- Seeding is idempotent and only runs when no visible products exist.
- Seed data is intended for local/dev previews and integration tests, not production.
- Fallback web catalog lives in `apps/web/src/lib/catalog-fallback.ts` and mirrors these vendors/categories.
