# Deployment Runbook

## CI pipeline
GitHub Actions runs on pull requests and on `main`:
- Web job: lint, typecheck, test, build.
- API job: go vet, go test, golangci-lint.

## Web deployment (Vercel)
1. Connect repository to Vercel project.
2. Set `MARKETPLACE_API_BASE_URL` to deployed API origin + `/api/v1`.
3. Deploy on `main` merge.

## API deployment (Render)
1. Build with `services/api/Dockerfile`.
2. Expose port `8080`.
3. Set required API env vars:
   - auth/JWT,
   - role bootstrap emails,
   - Stripe secrets/mode,
   - rate limit + request size settings.
4. Deploy from `main` branch.

## Post-deploy smoke checks
- `GET /health` returns `200`.
- Auth register/login/refresh works.
- Catalog list is reachable.
- Stripe webhook endpoint responds to signed test event.
- Admin/vendor protected endpoints reject unauthorized access.
