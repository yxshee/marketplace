# Release Verification Report (`v1.0.0`)

Date: 2026-02-08

## Scope
Final release verification for the multi-vendor marketplace across web, API, and shared contracts.

## Verification commands
```bash
pnpm -r lint
pnpm -r typecheck
pnpm -r test
pnpm -r build
cd services/api && go test ./...
pnpm audit --audit-level=high
```

## Results summary
- `pnpm -r lint` ✅ pass
- `pnpm -r typecheck` ✅ pass
- `pnpm -r test` ✅ pass
- `pnpm -r build` ✅ pass
- `go test ./...` (API) ✅ pass
- `pnpm audit --audit-level=high` ✅ no known vulnerabilities

## Go vulnerability audit note
A local `govulncheck` run using an older local Go toolchain reported standard-library CVEs tied to `go1.19.3`.

Actions taken:
- CI upgraded to Go `1.24` (`.github/workflows/ci.yml`).
- API Docker builder upgraded to Go `1.24` (`services/api/Dockerfile`).

Operational requirement:
- Keep build/deploy/runtime Go versions on patched 1.24.x releases.

## Release readiness checklist
- Buyer/Vendor/Admin milestone branches merged ✅
- Security hardening branch merged ✅
- Docs finalized (architecture, API, runbooks, seed data) ✅
- Tracking artifacts updated ✅
- CI green on final PR ✅

## Tagging
- Pre-release scaffold tag: `v0.0.0`
- Final release tag target: `v1.0.0`
