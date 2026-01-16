# Database Migration

Create and apply database migrations.

## Trigger
`/migrate <action>` or `/db`

## Actions

### Create Migration
`/migrate create <name>`

```bash
migrate create -ext sql -dir db/migrations -seq ${name}
```

This creates:
- `db/migrations/XXXXXX_${name}.up.sql`
- `db/migrations/XXXXXX_${name}.down.sql`

### Apply Migrations (Local)
`/migrate up`

```bash
make migrate-up
```

### Apply to Staging
`/migrate staging`

```bash
ssh -p 5022 blackpc@61.245.248.246 'cd /home/blackpc/saas-kerp && docker-compose exec api ./migrate -path db/migrations -database "$DATABASE_URL" up'
```

### Rollback
`/migrate down [steps]`

```bash
make migrate-down STEPS=${steps:-1}
```

### Status
`/migrate status`

```bash
migrate -path db/migrations -database "$DATABASE_URL" version
```

## Best Practices
1. Always create both up and down migrations
2. Test migrations locally first
3. Never modify applied migrations
4. Use transactions for data migrations
5. Include company_id in all business tables
