# sqlc queries

These queries define the foundation read/write contracts for:

- first-party auth (`users.sql`, `sessions.sql`)
- vendor onboarding and verification (`vendors.sql`)
- catalog and moderation workflow (`products.sql`)

Run generation from `services/api`:

```bash
sqlc generate
```
