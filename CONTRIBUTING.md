# Contributing

## Branching Rules
- `feat/<area>-<short-scope>`
- `fix/<area>-<short-scope>`
- `chore/<area>-<short-scope>`
- `docs/<area>-<short-scope>`

No direct commits to `main` after repository initialization.

## Commit Convention
Use Conventional Commits, e.g.:
- `feat(api): add shipment state machine`
- `fix(web): prevent duplicate checkout submission`
- `chore(ci): add api lint workflow`

## Required Verification Per PR
1. `pnpm -r lint`
2. `pnpm -r typecheck`
3. `pnpm -r test`
4. `pnpm -r build`
5. `go test ./...` in `services/api`

Include proof outputs in the PR body. Include screenshots for UI changes.

## Security Baseline
- Never commit secrets.
- Validate all external input.
- Enforce RBAC in API.
- Verify Stripe webhook signatures and idempotency.
