-- Initial Schema Migration
-- This is a consolidated migration that creates the complete database schema
-- It combines all previous migrations into a single clean initial setup

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Companies table - top level entity
CREATE TABLE "companies" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "name" VARCHAR(255) NOT NULL,
  "domain" VARCHAR(255) UNIQUE,
  "is_active" BOOLEAN DEFAULT TRUE,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE
);

-- 2. Users table - belongs to company
CREATE TABLE "users" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "company_id" UUID NOT NULL,
  "email" VARCHAR(255) NOT NULL,
  "password_hash" VARCHAR(255),
  "first_name" VARCHAR(255),
  "last_name" VARCHAR(255),
  "role" VARCHAR(50) DEFAULT 'user', -- 'admin' or 'user'
  "is_active" BOOLEAN DEFAULT TRUE,
  "invitation_token" VARCHAR(255) UNIQUE,
  "invitation_sent_at" TIMESTAMP WITH TIME ZONE,
  "invitation_accepted_at" TIMESTAMP WITH TIME ZONE,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE,
  CONSTRAINT "unique_company_email" UNIQUE ("company_id", "email"),
  CONSTRAINT "fk_users_company_id" 
    FOREIGN KEY ("company_id") REFERENCES "companies" ("id") ON DELETE CASCADE
);

-- 3. Apps table - defines available applications
CREATE TABLE "apps" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "name" VARCHAR(255) NOT NULL UNIQUE,
  "display_name" VARCHAR(255) NOT NULL,
  "description" TEXT,
  "type" VARCHAR(50) NOT NULL, -- 'source', 'destination', or 'both'
  "metadata_schema" JSONB NOT NULL DEFAULT '{}',
  "is_active" BOOLEAN DEFAULT TRUE,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE
);

-- 4. Connections table - OAuth credentials per company and app
CREATE TABLE "connections" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "user_id" UUID NOT NULL,
  "company_id" UUID NOT NULL,
  "app_id" UUID NOT NULL,
  "access_token" TEXT NOT NULL,
  "refresh_token" TEXT NOT NULL,
  "token_expires_at" TIMESTAMP WITH TIME ZONE,
  "provider_user_id" VARCHAR(255),
  "realm_id" VARCHAR(255), -- For QuickBooks Company ID
  "type" VARCHAR(50),
  "auth_type" VARCHAR(50) DEFAULT 'OAUTH2',
  "status" VARCHAR(50) DEFAULT 'ACTIVE',
  "is_active" BOOLEAN DEFAULT TRUE,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE,
  CONSTRAINT "fk_connections_user_id" 
    FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE,
  CONSTRAINT "fk_connections_company_id" 
    FOREIGN KEY ("company_id") REFERENCES "companies" ("id") ON DELETE CASCADE,
  CONSTRAINT "fk_connections_app_id" 
    FOREIGN KEY ("app_id") REFERENCES "apps" ("id") ON DELETE CASCADE
);

-- 5. Metadata table - app-specific configuration for connections
CREATE TABLE "metadata" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "connection_id" UUID NOT NULL UNIQUE,
  "data" JSONB NOT NULL DEFAULT '{}',
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  CONSTRAINT "fk_metadata_connection_id" 
    FOREIGN KEY ("connection_id") REFERENCES "connections" ("id") ON DELETE CASCADE
);

-- 6. Integrations table - sync workflows between source and destination apps (legacy, kept for backward compatibility)
CREATE TABLE "integrations" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "company_id" UUID NOT NULL,
  "name" VARCHAR(255) NOT NULL,
  "source_app_id" UUID NOT NULL,
  "destination_app_id" UUID NOT NULL,
  "source_connection_id" UUID,
  "destination_connection_id" UUID,
  "webhook_url" VARCHAR(500),
  "field_mapping" JSONB DEFAULT '{}',
  "sync_frequency_minutes" INTEGER DEFAULT 60,
  "status" VARCHAR(50) DEFAULT 'active',
  "last_synced_at" TIMESTAMP WITH TIME ZONE,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE,
  CONSTRAINT "fk_integrations_company_id" 
    FOREIGN KEY ("company_id") REFERENCES "companies" ("id") ON DELETE CASCADE,
  CONSTRAINT "fk_integrations_source_app_id" 
    FOREIGN KEY ("source_app_id") REFERENCES "apps" ("id") ON DELETE RESTRICT,
  CONSTRAINT "fk_integrations_destination_app_id" 
    FOREIGN KEY ("destination_app_id") REFERENCES "apps" ("id") ON DELETE RESTRICT,
  CONSTRAINT "fk_integrations_source_connection_id" 
    FOREIGN KEY ("source_connection_id") REFERENCES "connections" ("id") ON DELETE SET NULL,
  CONSTRAINT "fk_integrations_destination_connection_id" 
    FOREIGN KEY ("destination_connection_id") REFERENCES "connections" ("id") ON DELETE SET NULL
);

-- 7. Data Objects table - represents what exactly to read/write
CREATE TABLE "data_objects" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "connection_id" UUID NOT NULL,
  "object_type" VARCHAR(50) NOT NULL, -- 'SHEET', 'TABLE', 'INVOICE', 'ENTITY', 'OBJECT'
  "identifier" TEXT NOT NULL,
  "config" JSONB DEFAULT '{}',
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE,
  CONSTRAINT "fk_data_objects_connection_id" 
    FOREIGN KEY ("connection_id") REFERENCES "connections" ("id") ON DELETE CASCADE
);

-- 8. Pipelines table - source → destination mapping
CREATE TABLE "pipelines" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "company_id" UUID NOT NULL,
  "source_object_id" UUID NOT NULL,
  "destination_object_id" UUID NOT NULL,
  "sync_type" VARCHAR(50) DEFAULT 'PULL',
  "schedule_interval" INTEGER DEFAULT 60,
  "status" VARCHAR(50) DEFAULT 'ACTIVE',
  "field_mapping" JSONB DEFAULT '{}',
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE,
  CONSTRAINT "fk_pipelines_company_id" 
    FOREIGN KEY ("company_id") REFERENCES "companies" ("id") ON DELETE CASCADE,
  CONSTRAINT "fk_pipelines_source_object_id" 
    FOREIGN KEY ("source_object_id") REFERENCES "data_objects" ("id") ON DELETE RESTRICT,
  CONSTRAINT "fk_pipelines_destination_object_id" 
    FOREIGN KEY ("destination_object_id") REFERENCES "data_objects" ("id") ON DELETE RESTRICT
);

-- 9. Checkpoints table - guarantees no data loss
CREATE TABLE "checkpoints" (
  "pipeline_id" UUID PRIMARY KEY,
  "last_sync_time" TIMESTAMP WITH TIME ZONE NOT NULL,
  "last_cursor" TEXT,
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  CONSTRAINT "fk_checkpoints_pipeline_id" 
    FOREIGN KEY ("pipeline_id") REFERENCES "pipelines" ("id") ON DELETE CASCADE
);

-- 10. Sync Runs table - observability for each pipeline execution
CREATE TABLE "sync_runs" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "pipeline_id" UUID NOT NULL,
  "status" VARCHAR(50) NOT NULL,
  "records_read" INTEGER DEFAULT 0,
  "records_written" INTEGER DEFAULT 0,
  "error" TEXT,
  "started_at" TIMESTAMP WITH TIME ZONE NOT NULL,
  "finished_at" TIMESTAMP WITH TIME ZONE,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  CONSTRAINT "fk_sync_runs_pipeline_id" 
    FOREIGN KEY ("pipeline_id") REFERENCES "pipelines" ("id") ON DELETE CASCADE
);

-- Create indexes for performance
CREATE INDEX "idx_companies_domain" ON "companies" ("domain");
CREATE INDEX "idx_companies_deleted_at" ON "companies" ("deleted_at");
CREATE INDEX "idx_companies_is_active" ON "companies" ("is_active");

CREATE INDEX "idx_users_company_id" ON "users" ("company_id");
CREATE INDEX "idx_users_email" ON "users" ("email");
CREATE INDEX "idx_users_role" ON "users" ("role");
CREATE INDEX "idx_users_invitation_token" ON "users" ("invitation_token") WHERE "invitation_token" IS NOT NULL;
CREATE INDEX "idx_users_deleted_at" ON "users" ("deleted_at");

CREATE INDEX "idx_apps_name" ON "apps" ("name");
CREATE INDEX "idx_apps_type" ON "apps" ("type");
CREATE INDEX "idx_apps_is_active" ON "apps" ("is_active");
CREATE INDEX "idx_apps_deleted_at" ON "apps" ("deleted_at");

CREATE INDEX "idx_connections_user_id" ON "connections" ("user_id");
CREATE INDEX "idx_connections_company_id" ON "connections" ("company_id");
CREATE INDEX "idx_connections_app_id" ON "connections" ("app_id");
CREATE INDEX "idx_connections_is_active" ON "connections" ("is_active");
CREATE INDEX "idx_connections_deleted_at" ON "connections" ("deleted_at");

CREATE INDEX "idx_metadata_connection_id" ON "metadata" ("connection_id");

CREATE INDEX "idx_integrations_company_id" ON "integrations" ("company_id");
CREATE INDEX "idx_integrations_source_app_id" ON "integrations" ("source_app_id");
CREATE INDEX "idx_integrations_destination_app_id" ON "integrations" ("destination_app_id");
CREATE INDEX "idx_integrations_source_connection_id" ON "integrations" ("source_connection_id");
CREATE INDEX "idx_integrations_destination_connection_id" ON "integrations" ("destination_connection_id");
CREATE INDEX "idx_integrations_status" ON "integrations" ("status");
CREATE INDEX "idx_integrations_deleted_at" ON "integrations" ("deleted_at");

CREATE INDEX "idx_data_objects_connection_id" ON "data_objects" ("connection_id");
CREATE INDEX "idx_data_objects_object_type" ON "data_objects" ("object_type");
CREATE INDEX "idx_data_objects_deleted_at" ON "data_objects" ("deleted_at");

CREATE INDEX "idx_pipelines_company_id" ON "pipelines" ("company_id");
CREATE INDEX "idx_pipelines_source_object_id" ON "pipelines" ("source_object_id");
CREATE INDEX "idx_pipelines_destination_object_id" ON "pipelines" ("destination_object_id");
CREATE INDEX "idx_pipelines_status" ON "pipelines" ("status");
CREATE INDEX "idx_pipelines_deleted_at" ON "pipelines" ("deleted_at");

CREATE INDEX "idx_sync_runs_pipeline_id" ON "sync_runs" ("pipeline_id");
CREATE INDEX "idx_sync_runs_status" ON "sync_runs" ("status");
CREATE INDEX "idx_sync_runs_started_at" ON "sync_runs" ("started_at");

-- Seed initial apps
INSERT INTO "apps" ("id", "name", "display_name", "description", "type", "metadata_schema", "is_active") VALUES
(
  uuid_generate_v4(),
  'googlesheet',
  'Google Sheets',
  'Sync data from Google Sheets',
  'source',
  '{
    "required_fields": ["filename", "sheetname"],
    "optional_fields": ["url", "range"],
    "field_descriptions": {
      "filename": "Name of the Google Sheet file",
      "sheetname": "Name of the specific sheet/tab within the file",
      "url": "Full URL of the Google Sheet (optional)",
      "range": "Cell range to sync (e.g., A1:Z100) (optional)"
    }
  }'::jsonb,
  true
),
(
  uuid_generate_v4(),
  'salesforce',
  'Salesforce',
  'Sync data to/from Salesforce CRM',
  'both',
  '{
    "required_fields": ["instance_url", "api_version"],
    "optional_fields": ["org_id", "username"],
    "field_descriptions": {
      "instance_url": "Salesforce instance URL (e.g., https://yourinstance.salesforce.com)",
      "api_version": "Salesforce API version (e.g., v57.0)",
      "org_id": "Salesforce Organization ID (optional)",
      "username": "Salesforce username (optional)"
    }
  }'::jsonb,
  true
),
(
  uuid_generate_v4(),
  'quickbooks',
  'QuickBooks',
  'Sync data to/from QuickBooks accounting software',
  'both',
  '{
    "required_fields": ["realm_id"],
    "optional_fields": ["company_name"],
    "field_descriptions": {
      "realm_id": "QuickBooks Company ID (Realm ID)",
      "company_name": "QuickBooks company name (optional)"
    }
  }'::jsonb,
  true
),
(
  uuid_generate_v4(),
  'tally',
  'Tally',
  'Sync data to Tally accounting software',
  'destination',
  '{
    "required_fields": ["server_url", "port", "company_name"],
    "optional_fields": ["username", "password"],
    "field_descriptions": {
      "server_url": "Tally server URL",
      "port": "Tally server port number",
      "company_name": "Tally company name",
      "username": "Tally username (optional)",
      "password": "Tally password (optional)"
    }
  }'::jsonb,
  true
),
(
  uuid_generate_v4(),
  'googledrive',
  'Google Drive',
  'Sync files from Google Drive',
  'source',
  '{
    "required_fields": ["file_id"],
    "optional_fields": ["file_name", "mime_type", "folder_id"],
    "field_descriptions": {
      "file_id": "Google Drive file ID",
      "file_name": "Name of the file (optional)",
      "mime_type": "MIME type of the file (optional)",
      "folder_id": "Google Drive folder ID (optional)"
    }
  }'::jsonb,
  true
);

