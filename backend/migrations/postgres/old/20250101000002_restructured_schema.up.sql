-- Restructured Schema Migration
-- This migration creates the new schema structure as per requirements

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

-- 2. Users table - belongs to company, one admin per company
CREATE TABLE "users" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "company_id" UUID NOT NULL,
  "email" VARCHAR(255) NOT NULL,
  "password_hash" VARCHAR(255),
  "first_name" VARCHAR(255),
  "last_name" VARCHAR(255),
  "role" VARCHAR(50) DEFAULT 'user', -- 'admin' or 'user'
  "is_active" BOOLEAN DEFAULT TRUE,
  "invitation_token" VARCHAR(255) UNIQUE, -- For email invitations
  "invitation_sent_at" TIMESTAMP WITH TIME ZONE,
  "invitation_accepted_at" TIMESTAMP WITH TIME ZONE,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE,
  CONSTRAINT "unique_company_email" UNIQUE ("company_id", "email")
);

-- 3. Apps table - defines available applications (Salesforce, GoogleSheet, etc.)
CREATE TABLE "apps" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "name" VARCHAR(255) NOT NULL UNIQUE, -- e.g., "salesforce", "googlesheet"
  "display_name" VARCHAR(255) NOT NULL, -- e.g., "Salesforce", "Google Sheets"
  "description" TEXT,
  "type" VARCHAR(50) NOT NULL, -- 'source', 'destination', or 'both'
  "metadata_schema" JSONB NOT NULL DEFAULT '{}', -- Schema defining required metadata fields
  -- Example: {"required_fields": ["filename", "sheetname"], "optional_fields": ["url"]}
  "is_active" BOOLEAN DEFAULT TRUE,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE
);

-- 4. Connections table - OAuth credentials per company and app
CREATE TABLE "connections" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "company_id" UUID NOT NULL,
  "app_id" UUID NOT NULL,
  "access_token" TEXT NOT NULL, -- Encrypted at application level
  "refresh_token" TEXT NOT NULL, -- Encrypted at application level
  "token_expires_at" TIMESTAMP WITH TIME ZONE,
  "provider_user_id" VARCHAR(255), -- User ID from the app provider
  "realm_id" VARCHAR(255), -- For QuickBooks Company ID
  "is_active" BOOLEAN DEFAULT TRUE,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE,
  CONSTRAINT "unique_company_app_connection" UNIQUE ("company_id", "app_id")
);

-- 5. Metadata table - app-specific configuration for connections
CREATE TABLE "metadata" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "connection_id" UUID NOT NULL UNIQUE, -- One-to-one with connection
  "data" JSONB NOT NULL DEFAULT '{}', -- Stores app-specific metadata
  -- Example for GoogleSheet: {"filename": "sheet.xlsx", "sheetname": "Sheet1", "url": "https://..."}
  -- Example for Salesforce: {"instance_url": "https://...", "api_version": "v57.0"}
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 6. Integrations table - sync workflows between source and destination apps
CREATE TABLE "integrations" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "company_id" UUID NOT NULL,
  "name" VARCHAR(255) NOT NULL,
  "source_app_id" UUID NOT NULL, -- References apps table
  "destination_app_id" UUID NOT NULL, -- References apps table
  "source_connection_id" UUID, -- References connections table
  "destination_connection_id" UUID, -- References connections table
  "webhook_url" VARCHAR(500), -- Optional webhook URL for real-time sync
  "field_mapping" JSONB DEFAULT '{}', -- Field mapping configuration
  "sync_frequency_minutes" INTEGER DEFAULT 60,
  "status" VARCHAR(50) DEFAULT 'active', -- 'active', 'paused', 'error'
  "last_synced_at" TIMESTAMP WITH TIME ZONE,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE
);

-- Create indexes for performance
CREATE INDEX "idx_companies_domain" ON "companies" ("domain");
CREATE INDEX "idx_companies_deleted_at" ON "companies" ("deleted_at");
CREATE INDEX "idx_companies_is_active" ON "companies" ("is_active");

CREATE INDEX "idx_users_company_id" ON "users" ("company_id");
CREATE INDEX "idx_users_email" ON "users" ("email");
CREATE INDEX "idx_users_role" ON "users" ("role");
CREATE INDEX "idx_users_invitation_token" ON "users" ("invitation_token");
CREATE INDEX "idx_users_deleted_at" ON "users" ("deleted_at");

CREATE INDEX "idx_apps_name" ON "apps" ("name");
CREATE INDEX "idx_apps_type" ON "apps" ("type");
CREATE INDEX "idx_apps_is_active" ON "apps" ("is_active");
CREATE INDEX "idx_apps_deleted_at" ON "apps" ("deleted_at");

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

-- Add foreign key constraints
ALTER TABLE "users" ADD CONSTRAINT "fk_users_company_id" 
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id") ON DELETE CASCADE;

ALTER TABLE "connections" ADD CONSTRAINT "fk_connections_company_id" 
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id") ON DELETE CASCADE;

ALTER TABLE "connections" ADD CONSTRAINT "fk_connections_app_id" 
  FOREIGN KEY ("app_id") REFERENCES "apps" ("id") ON DELETE CASCADE;

ALTER TABLE "metadata" ADD CONSTRAINT "fk_metadata_connection_id" 
  FOREIGN KEY ("connection_id") REFERENCES "connections" ("id") ON DELETE CASCADE;

ALTER TABLE "integrations" ADD CONSTRAINT "fk_integrations_company_id" 
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id") ON DELETE CASCADE;

ALTER TABLE "integrations" ADD CONSTRAINT "fk_integrations_source_app_id" 
  FOREIGN KEY ("source_app_id") REFERENCES "apps" ("id") ON DELETE RESTRICT;

ALTER TABLE "integrations" ADD CONSTRAINT "fk_integrations_destination_app_id" 
  FOREIGN KEY ("destination_app_id") REFERENCES "apps" ("id") ON DELETE RESTRICT;

ALTER TABLE "integrations" ADD CONSTRAINT "fk_integrations_source_connection_id" 
  FOREIGN KEY ("source_connection_id") REFERENCES "connections" ("id") ON DELETE SET NULL;

ALTER TABLE "integrations" ADD CONSTRAINT "fk_integrations_destination_connection_id" 
  FOREIGN KEY ("destination_connection_id") REFERENCES "connections" ("id") ON DELETE SET NULL;

-- Add constraint to ensure at least one admin per company
-- This will be enforced at application level, but we can add a check constraint
-- Note: This is complex to enforce at DB level, so it's better handled in application logic




