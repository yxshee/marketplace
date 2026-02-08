# feat/admin-audit-logs

Status: Ready for review.

## Implemented scope
- Added admin audit log listing API endpoint with RBAC enforcement.
- Added shared API contracts/schemas and web API client support for audit log queries.
- Integrated admin audit log viewer section on admin surface.
- Instrumented key admin mutation endpoints to persist structured audit entries.
- Added backend/service/router tests for audit logging and authorization behavior.

## Completion checklist
- [x] Implementation complete
- [x] pnpm -r lint
- [x] pnpm -r typecheck
- [x] pnpm -r test
- [x] pnpm -r build
- [x] go test ./... (API)
- [x] UI screenshots included (if UI changes)
- [x] Proof outputs added to PR description
