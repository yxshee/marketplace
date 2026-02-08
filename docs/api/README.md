# API Documentation

## Source of truth
- OpenAPI specification: `services/api/openapi/openapi.yaml`
- Shared TS contracts: `packages/shared/src/contracts/api.ts`
- API client bindings: `apps/web/src/lib/api-client.ts`

## References
- Endpoint grouping and responsibilities: `docs/api/endpoints.md`
- RBAC mapping: `docs/architecture/rbac-matrix.md`

## Notes
- API is versioned under `/api/v1`.
- Business rules are enforced in API handlers/services, not only in UI.
- List endpoints include pagination bounds and metadata (`limit`, `offset`, `total`).
