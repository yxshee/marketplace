# Runbooks

## Available runbooks
- `deployment.md` - CI and deployment flow for web/API.
- `seed-data.md` - Development seed catalog and vendor data behavior.
- `release-v1.0.0-verification.md` - Final verification and audit evidence for release.

## Operational principles
- Use API as source of truth for order/payment/refund state transitions.
- Prefer replay-safe and idempotent operations for payment workflows.
- Keep PR evidence attached (commands + outputs + screenshots for UI changes).
