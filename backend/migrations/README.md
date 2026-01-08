# Database Migrations

This directory contains PostgreSQL migration files for the syncflow-backend database.

## Migration Files

### 20250101000000_init_schema
Creates the core schema for the new architecture:
- `users` - Tenant/user management
- `connections` - OAuth credentials storage
- `integrations` - Sync workflow configuration
- `sync_jobs` - Sync execution history
- `sync_records` - Granular sync logs with retry support

### 20250101000001_legacy_tables
Creates legacy tables for backward compatibility:
- `companies` - Legacy company model
- `legacy_integrations` - Legacy integration model
- `field_mappings` - Field mapping configuration
- `legacy_sync_records` - Legacy sync record tracking
- `sync_logs` - Sync operation logs
- `tally_configs` - Tally integration configuration

## Running Migrations

### Using golang-migrate

```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations up
migrate -path migrations/postgres -database "postgres://user:password@localhost/dbname?sslmode=disable" up

# Run migrations down
migrate -path migrations/postgres -database "postgres://user:password@localhost/dbname?sslmode=disable" down

# Check migration version
migrate -path migrations/postgres -database "postgres://user:password@localhost/dbname?sslmode=disable" version
```

### Using psql

```bash
# Run all up migrations
psql -U postgres -d syncflow -f migrations/postgres/20250101000000_init_schema.up.sql
psql -U postgres -d syncflow -f migrations/postgres/20250101000001_legacy_tables.up.sql

# Rollback (run down migrations in reverse order)
psql -U postgres -d syncflow -f migrations/postgres/20250101000001_legacy_tables.down.sql
psql -U postgres -d syncflow -f migrations/postgres/20250101000000_init_schema.down.sql
```

## Migration Naming Convention

Migrations follow the pattern: `YYYYMMDDHHMMSS_description.up.sql` and `YYYYMMDDHHMMSS_description.down.sql`

- `up.sql` - Contains SQL to apply the migration
- `down.sql` - Contains SQL to rollback the migration

## Notes

- UUID extension (`uuid-ossp`) is enabled in the init migration
- All timestamps use `TIMESTAMP WITH TIME ZONE`
- Foreign keys use `ON DELETE CASCADE` for related records
- Indexes are created for frequently queried columns
- Soft deletes are supported via `deleted_at` columns





