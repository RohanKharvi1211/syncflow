# Database Migration Guide

## Overview
This guide explains the database restructuring that has been implemented.

## New Schema Structure

### Key Changes

1. **Company-Centric Model**: Companies are now the top-level entity
2. **User Management**: Users belong to companies, with role-based access (admin/user)
3. **App Registry**: Centralized app definitions with metadata schemas
4. **Connection Model**: Connections link companies to apps (not users)
5. **Metadata Separation**: App-specific configuration moved to separate metadata table
6. **Integration Updates**: Integrations reference apps and connections

## Migration Files

### 1. `20250101000002_restructured_schema.up.sql`
Creates the new schema structure:
- `companies` table
- `users` table (with company_id, role, invitation fields)
- `apps` table (with metadata_schema)
- `connections` table (with company_id, app_id)
- `metadata` table (one-to-one with connections)
- `integrations` table (with source_app_id, destination_app_id, webhook_url)

### 2. `20250101000003_seed_apps.up.sql`
Seeds initial apps:
- Google Sheets (source)
- Salesforce (both)
- QuickBooks (destination)
- Tally (destination)
- Google Drive (source)

## Running Migrations

### Up Migration
```bash
# Run the migration
migrate -path ./migrations/postgres -database "postgres://user:pass@localhost/dbname?sslmode=disable" up
```

### Down Migration (Rollback)
```bash
# Rollback the migration
migrate -path ./migrations/postgres -database "postgres://user:pass@localhost/dbname?sslmode=disable" down
```

## Model Updates

All models have been updated in:
- `backend/internal/models/company.go` - Company model (now uses UUID)
- `backend/internal/models/user.go` - User, App, Connection, Metadata, Integration models

## Key Features

### 1. User Invitations
- `invitation_token` field for email invitations
- `invitation_sent_at` and `invitation_accepted_at` for tracking

### 2. App Metadata Schema
Each app defines its required metadata fields:
```json
{
  "required_fields": ["filename", "sheetname"],
  "optional_fields": ["url"],
  "field_descriptions": {
    "filename": "Name of the Google Sheet file"
  }
}
```

### 3. Metadata Validation
Use `backend/internal/utils/metadata_validator.go` to validate metadata against app schemas.

## Authorization

### Admin Permissions
- Create/update/delete apps
- Invite users
- Manage company settings
- Delete company

### User Permissions
- Create connections
- Create integrations
- View company data
- Update own profile

## Next Steps

1. **Update Controllers**: Update API controllers to use new models
2. **Update Services**: Update business logic to work with new schema
3. **Add Authorization Middleware**: Implement role-based access control
4. **Email Service**: Implement user invitation email sending
5. **Metadata Validation**: Integrate metadata validator in connection creation

## Backward Compatibility

The old schema tables are preserved as "legacy" tables:
- `legacy_integrations`
- `legacy_sync_records`
- Old `companies` table (with integer ID)

You may need to create a data migration script to move data from old to new schema if needed.




