-- Legacy tables for backward compatibility during migration

-- Create companies table (legacy)
CREATE TABLE "companies" (
  "id" SERIAL PRIMARY KEY,
  "name" VARCHAR(255) NOT NULL,
  "domain" VARCHAR(255) UNIQUE,
  "is_active" BOOLEAN DEFAULT TRUE,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE
);

-- Create legacy_integrations table
CREATE TABLE "legacy_integrations" (
  "id" SERIAL PRIMARY KEY,
  "company_id" INTEGER NOT NULL,
  "type" VARCHAR(50),
  "name" VARCHAR(255),
  "is_active" BOOLEAN DEFAULT TRUE,
  "config" TEXT,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create field_mappings table
CREATE TABLE "field_mappings" (
  "id" SERIAL PRIMARY KEY,
  "company_id" INTEGER NOT NULL,
  "source_system" VARCHAR(50),
  "target_system" VARCHAR(50),
  "source_object" VARCHAR(100),
  "target_object" VARCHAR(100),
  "source_field" VARCHAR(255),
  "target_field" VARCHAR(255),
  "mapping_type" VARCHAR(50),
  "transform_rule" TEXT,
  "is_required" BOOLEAN DEFAULT FALSE,
  "is_active" BOOLEAN DEFAULT TRUE,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create legacy_sync_records table
CREATE TABLE "legacy_sync_records" (
  "id" SERIAL PRIMARY KEY,
  "company_id" INTEGER NOT NULL,
  "source_record_id" VARCHAR(255) NOT NULL,
  "target_record_id" VARCHAR(255),
  "source_system" VARCHAR(50),
  "target_system" VARCHAR(50),
  "record_type" VARCHAR(100),
  "status" VARCHAR(50),
  "error_message" TEXT,
  "last_sync_attempt" TIMESTAMP WITH TIME ZONE,
  "last_successful_sync" TIMESTAMP WITH TIME ZONE,
  "retry_count" INTEGER DEFAULT 0,
  "source_data" TEXT,
  "target_data" TEXT,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE
);

-- Create sync_logs table
CREATE TABLE "sync_logs" (
  "id" SERIAL PRIMARY KEY,
  "company_id" INTEGER NOT NULL,
  "sync_record_id" INTEGER,
  "action" VARCHAR(50),
  "status" VARCHAR(50),
  "message" TEXT,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create tally_configs table
CREATE TABLE "tally_configs" (
  "id" SERIAL PRIMARY KEY,
  "company_id" INTEGER NOT NULL,
  "server_url" VARCHAR(255),
  "port" INTEGER,
  "username" VARCHAR(255),
  "password" VARCHAR(255),
  "company_name" VARCHAR(255),
  "is_active" BOOLEAN DEFAULT TRUE,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for legacy tables
CREATE INDEX "idx_companies_domain" ON "companies" ("domain");
CREATE INDEX "idx_companies_deleted_at" ON "companies" ("deleted_at");

CREATE INDEX "idx_legacy_integrations_company_id" ON "legacy_integrations" ("company_id");
CREATE INDEX "idx_legacy_integrations_type" ON "legacy_integrations" ("type");

CREATE INDEX "idx_field_mappings_company_id" ON "field_mappings" ("company_id");
CREATE INDEX "idx_field_mappings_source_target" ON "field_mappings" ("source_system", "target_system");

CREATE INDEX "idx_legacy_sync_records_company_id" ON "legacy_sync_records" ("company_id");
CREATE INDEX "idx_legacy_sync_records_status" ON "legacy_sync_records" ("status");
CREATE INDEX "idx_legacy_sync_records_deleted_at" ON "legacy_sync_records" ("deleted_at");

CREATE INDEX "idx_sync_logs_company_id" ON "sync_logs" ("company_id");
CREATE INDEX "idx_sync_logs_sync_record_id" ON "sync_logs" ("sync_record_id");

CREATE INDEX "idx_tally_configs_company_id" ON "tally_configs" ("company_id");

-- Add foreign key constraints
ALTER TABLE "legacy_integrations" ADD CONSTRAINT "fk_legacy_integrations_company_id" 
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id") ON DELETE CASCADE;

ALTER TABLE "field_mappings" ADD CONSTRAINT "fk_field_mappings_company_id" 
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id") ON DELETE CASCADE;

ALTER TABLE "legacy_sync_records" ADD CONSTRAINT "fk_legacy_sync_records_company_id" 
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id") ON DELETE CASCADE;

ALTER TABLE "sync_logs" ADD CONSTRAINT "fk_sync_logs_company_id" 
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id") ON DELETE CASCADE;

ALTER TABLE "sync_logs" ADD CONSTRAINT "fk_sync_logs_sync_record_id" 
  FOREIGN KEY ("sync_record_id") REFERENCES "legacy_sync_records" ("id") ON DELETE CASCADE;

ALTER TABLE "tally_configs" ADD CONSTRAINT "fk_tally_configs_company_id" 
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id") ON DELETE CASCADE;





