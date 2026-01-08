# Database Schema Documentation

## Overview
This document describes the restructured database schema for the data transfer application.

## Entity Relationships

```
Company (1) ──< (Many) Users
Company (1) ──< (Many) Connections
Company (1) ──< (Many) Integrations
App (1) ──< (Many) Connections
Connection (1) ──< (1) Metadata
Integration (Many) ──> (1) App (source)
Integration (Many) ──> (1) App (destination)
Integration (Many) ──> (1) Connection (source, optional)
Integration (Many) ──> (1) Connection (destination, optional)
```

## Tables

### 1. Companies
**Purpose**: Top-level entity representing a tenant/organization.

**Fields**:
- `id` (UUID) - Primary key
- `name` (VARCHAR) - Company name
- `domain` (VARCHAR) - Unique domain identifier
- `is_active` (BOOLEAN) - Active status
- `created_at`, `updated_at`, `deleted_at` - Timestamps

**Relationships**:
- Has many Users (one must be admin)
- Has many Connections
- Has many Integrations

**Business Rules**:
- At least one user must be an admin (enforced at application level)
- Company is created first, then admin user is created

---

### 2. Users
**Purpose**: Users belonging to a company. One admin per company.

**Fields**:
- `id` (UUID) - Primary key
- `company_id` (UUID) - Foreign key to companies
- `email` (VARCHAR) - User email (unique per company)
- `password_hash` (VARCHAR) - Hashed password
- `first_name`, `last_name` (VARCHAR) - User name
- `role` (VARCHAR) - 'admin' or 'user' (default: 'user')
- `is_active` (BOOLEAN) - Active status
- `invitation_token` (VARCHAR) - Token for email invitations
- `invitation_sent_at`, `invitation_accepted_at` (TIMESTAMP) - Invitation tracking
- `created_at`, `updated_at`, `deleted_at` - Timestamps

**Relationships**:
- Belongs to Company

**Business Rules**:
- Email must be unique within a company (composite unique constraint)
- At least one admin user per company
- Invitation token is generated when inviting a user
- Password is set when user accepts invitation

---

### 3. Apps
**Purpose**: Defines available applications (Salesforce, GoogleSheet, etc.) and their metadata requirements.

**Fields**:
- `id` (UUID) - Primary key
- `name` (VARCHAR) - Unique identifier (e.g., "salesforce", "googlesheet")
- `display_name` (VARCHAR) - Display name (e.g., "Salesforce", "Google Sheets")
- `description` (TEXT) - App description
- `type` (VARCHAR) - 'source', 'destination', or 'both'
- `metadata_schema` (JSONB) - Schema defining required/optional metadata fields
- `is_active` (BOOLEAN) - Active status
- `created_at`, `updated_at`, `deleted_at` - Timestamps

**Relationships**:
- Has many Connections
- Has many Integrations (as source)
- Has many Integrations (as destination)

**Metadata Schema Structure**:
```json
{
  "required_fields": ["filename", "sheetname"],
  "optional_fields": ["url", "range"],
  "field_descriptions": {
    "filename": "Name of the Google Sheet file",
    "sheetname": "Name of the specific sheet/tab"
  }
}
```

**Business Rules**:
- When creating a connection for an app, metadata must include all required_fields
- Validation happens at application level before saving metadata

---

### 4. Connections
**Purpose**: OAuth credentials linking a company to an app.

**Fields**:
- `id` (UUID) - Primary key
- `company_id` (UUID) - Foreign key to companies
- `app_id` (UUID) - Foreign key to apps
- `access_token` (TEXT) - OAuth access token (encrypted)
- `refresh_token` (TEXT) - OAuth refresh token (encrypted)
- `token_expires_at` (TIMESTAMP) - Token expiration
- `provider_user_id` (VARCHAR) - User ID from app provider
- `realm_id` (VARCHAR) - For QuickBooks Company ID
- `is_active` (BOOLEAN) - Active status
- `created_at`, `updated_at`, `deleted_at` - Timestamps

**Relationships**:
- Belongs to Company
- Belongs to App
- Has one Metadata (one-to-one)

**Business Rules**:
- One connection per company-app combination (unique constraint)
- Tokens are encrypted at application level
- Metadata must be created when connection is created

---

### 5. Metadata
**Purpose**: App-specific configuration for a connection.

**Fields**:
- `id` (UUID) - Primary key
- `connection_id` (UUID) - Foreign key to connections (unique)
- `data` (JSONB) - App-specific metadata

**Relationships**:
- Belongs to Connection (one-to-one)

**Metadata Examples**:

**Google Sheet**:
```json
{
  "filename": "SalesData.xlsx",
  "sheetname": "Sheet1",
  "url": "https://docs.google.com/spreadsheets/d/..."
}
```

**Salesforce**:
```json
{
  "instance_url": "https://yourinstance.salesforce.com",
  "api_version": "v57.0",
  "org_id": "00D..."
}
```

**Tally**:
```json
{
  "server_url": "http://localhost",
  "port": 9000,
  "company_name": "My Company",
  "username": "admin",
  "password": "password"
}
```

**Business Rules**:
- Must validate against app's metadata_schema before saving
- All required_fields from app.metadata_schema must be present
- Optional fields can be omitted

---

### 6. Integrations
**Purpose**: Sync workflows between source and destination apps.

**Fields**:
- `id` (UUID) - Primary key
- `company_id` (UUID) - Foreign key to companies
- `name` (VARCHAR) - Integration name
- `source_app_id` (UUID) - Foreign key to apps (source)
- `destination_app_id` (UUID) - Foreign key to apps (destination)
- `source_connection_id` (UUID) - Foreign key to connections (optional)
- `destination_connection_id` (UUID) - Foreign key to connections (optional)
- `webhook_url` (VARCHAR) - Optional webhook URL for real-time sync
- `field_mapping` (JSONB) - Field mapping configuration
- `sync_frequency_minutes` (INTEGER) - Sync frequency (default: 60)
- `status` (VARCHAR) - 'active', 'paused', 'error' (default: 'active')
- `last_synced_at` (TIMESTAMP) - Last sync timestamp
- `created_at`, `updated_at`, `deleted_at` - Timestamps

**Relationships**:
- Belongs to Company
- Belongs to App (as source)
- Belongs to App (as destination)
- Belongs to Connection (source, optional)
- Belongs to Connection (destination, optional)

**Business Rules**:
- Source and destination apps must be different
- Connections are optional (can be set up later)
- Field mapping defines how source fields map to destination fields

---

## Authorization Flow

### 1. Company Creation
1. Create company record
2. Create admin user for the company
3. Admin user can invite other users

### 2. User Invitation
1. Admin generates invitation token
2. Email sent with invitation link
3. User clicks link, sets password
4. `invitation_accepted_at` is set

### 3. App Management
1. Admin creates/updates apps
2. When creating app, define metadata_schema
3. Validation ensures required_fields are specified

### 4. Connection Creation
1. User initiates OAuth flow for an app
2. OAuth callback creates connection
3. User must provide metadata matching app's metadata_schema
4. Validation ensures all required_fields are present
5. Metadata record is created

### 5. Integration Creation
1. User selects source and destination apps
2. Optionally links connections
3. Configures field mapping
4. Sets webhook URL if needed

---

## Validation Rules

### Metadata Validation
When creating/updating metadata for a connection:
1. Fetch the app's metadata_schema
2. Check that all required_fields are present in metadata.data
3. Optional fields can be omitted
4. Reject if required fields are missing

### App Type Validation
When creating an integration:
1. Source app must have type 'source' or 'both'
2. Destination app must have type 'destination' or 'both'
3. Source and destination apps must be different

### User Role Validation
- Only admins can:
  - Invite users
  - Create/update apps
  - Delete company
- All users can:
  - Create connections
  - Create integrations
  - View their company's data

---

## Example Workflow

1. **Setup Company**
   ```
   POST /api/companies
   → Creates company
   → Creates admin user
   ```

2. **Invite User**
   ```
   POST /api/users/invite
   → Generates invitation_token
   → Sends email
   ```

3. **Create App**
   ```
   POST /api/apps
   {
     "name": "googlesheet",
     "metadata_schema": {
       "required_fields": ["filename", "sheetname"]
     }
   }
   ```

4. **Create Connection**
   ```
   POST /api/oauth/google/initiate
   → OAuth flow
   → POST /api/connections
   → POST /api/metadata (validates against app schema)
   ```

5. **Create Integration**
   ```
   POST /api/integrations
   {
     "source_app_id": "...",
     "destination_app_id": "...",
     "source_connection_id": "...",
     "destination_connection_id": "..."
   }
   ```




