# feat/buyer-payments-stripe

Status: Complete - recreated from origin/main and hardened Stripe/COD payment settings enforcement.

## Planned scope
- Implement the branch scope defined in the marketplace execution plan.
- Keep changes small, strongly typed, tested, and production-safe.

## Implemented in this branch
- Added `GET /api/v1/payments/settings` so checkout can render payment options from API truth.
- Updated Stripe intent creation to allow fresh intent creation after `payment_failed` retries.
- Added API/router/service tests for buyer payment settings and failed-payment retry behavior.
- Updated checkout UI to hide/disable methods based on admin settings and show clear fallback messages.

## Completion checklist
- [x] Implementation complete
- [x] pnpm -r lint
- [x] pnpm -r typecheck
- [x] pnpm -r test
- [x] pnpm -r build
- [x] go test ./... (API)
- [x] UI screenshots included (if UI changes)
- [x] Proof outputs added to PR description

## Screenshot proof
- `docs/screenshots/step4c-buyer-checkout-payment-settings.png`
