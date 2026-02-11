# Backend Verification Baseline (2026-02-11)

## Scope
- Branch: `chore_backend_verification`
- Repo: `/Users/venom/github/marketplace`
- Goal: establish reproducible pre-change baseline before backend audit/remediation.

## Key docs read and summarized
- `docs/architecture/architecture-plan.md`
- `docs/architecture/system-overview.md`
- `docs/architecture/rbac-matrix.md`
- `docs/api/endpoints.md`
- `docs/api/README.md`
- `docs/runbooks/deployment.md`
- `docs/runbooks/seed-data.md`
- `services/api/openapi/openapi.yaml`

## Confirmed invariants from docs/spec
- REST base path is `/api/v1` (OpenAPI `servers` section).
- Go API is the source of truth for business and authorization invariants.
- Checkout split invariant: one order, one shipment per vendor.
- Checkout/payment mutation requests include idempotency keys (spec + architecture docs).
- RBAC is enforced server-side (middleware + ownership checks).
- Stripe webhook uses signature verification and idempotent processing.
- S3-compatible object storage uses presigned upload flow.
- Seed data is development-only (`API_ENV=development`) and idempotent.

## Baseline quality gates

### Workspace install
- `pnpm install`: ✅ pass

### Requested workspace commands (as provided)
- `pnpm recursive lint`: ❌ fail (CLI usage)
- `pnpm recursive typecheck`: ❌ fail (CLI usage)
- `pnpm recursive test`: ❌ fail (CLI usage)
- `pnpm recursive build`: ❌ fail (CLI usage)

Error snippet:
```text
Usage: pnpm recursive [command] [flags] ...
Commands: ... run <command> ... test ...
```

Suspected root cause:
- This pnpm version expects `pnpm -r run <script>` or `pnpm recursive run <script>`, not `pnpm recursive <script>`.

### Equivalent workspace checks (for actual baseline signal)
- `pnpm -r run lint`: ✅ pass
- `pnpm -r run typecheck`: ✅ pass
- `pnpm -r run test`: ✅ pass
- `pnpm -r run build`: ✅ pass

## API-specific checks
- `cd services/api && go test ./...`: ✅ pass
- `cd services/api && make test`: ✅ pass
- `cd services/api && make lint`: ❌ fail

Error snippet:
```text
golangci-lint run ./...
make: golangci-lint: No such file or directory
```

Suspected root cause:
- Local developer toolchain missing `golangci-lint` binary.

- `cd services/api && make sqlc_generate`: ❌ fail

Error snippet:
```text
make: *** No rule to make target `sqlc_generate'.  Stop.
```

Suspected root cause:
- Target name mismatch; Makefile defines `sqlc-generate` (hyphen), not `sqlc_generate`.

- `cd services/api && make sqlc-generate`: ❌ fail

Error snippet:
```text
sqlc generate
make: sqlc: No such file or directory
```

Suspected root cause:
- Local developer toolchain missing `sqlc` binary.

## Baseline outcome
- Application/test code baseline appears healthy for existing configured checks.
- Blocking gaps are local tooling availability (`golangci-lint`, `sqlc`) and pnpm command syntax mismatch.
