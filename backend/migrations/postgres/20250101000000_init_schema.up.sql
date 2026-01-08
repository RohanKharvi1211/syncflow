-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create users table (tenants)
CREATE TABLE "users" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "email" VARCHAR(255) UNIQUE NOT NULL,
  "password_hash" VARCHAR(255),
  "company_name" VARCHAR(255),
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE
);

-- Create connections table (OAuth credentials)
CREATE TABLE "connections" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "user_id" UUID NOT NULL,
  "provider" VARCHAR(50) NOT NULL,
  "provider_user_id" VARCHAR(255),
  "access_token" TEXT NOT NULL,
  "refresh_token" TEXT NOT NULL,
  "token_expires_at" TIMESTAMP WITH TIME ZONE,
  "realm_id" VARCHAR(255),
  "metadata" JSONB DEFAULT '{}',
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE
);

-- Create integrations table (sync workflow configuration)
CREATE TABLE "integrations" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "user_id" UUID NOT NULL,
  "name" VARCHAR(255) NOT NULL,
  "source_connection_id" UUID,
  "destination_connection_id" UUID,
  "source_file_id" VARCHAR(255),
  "source_sheet_name" VARCHAR(255),
  "source_primary_key" VARCHAR(255),
  "destination_entity" VARCHAR(100),
  "field_mapping" JSONB NOT NULL DEFAULT '{}',
  "sync_frequency_minutes" INTEGER DEFAULT 60,
  "status" VARCHAR(50) DEFAULT 'active',
  "last_synced_at" TIMESTAMP WITH TIME ZONE,
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE
);

-- Create sync_jobs table (execution history)
CREATE TABLE "sync_jobs" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "integration_id" UUID NOT NULL,
  "start_time" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "end_time" TIMESTAMP WITH TIME ZONE,
  "status" VARCHAR(50) DEFAULT 'processing',
  "records_processed" INTEGER DEFAULT 0,
  "records_failed" INTEGER DEFAULT 0,
  "records_success" INTEGER DEFAULT 0,
  "trigger_type" VARCHAR(50) DEFAULT 'scheduled',
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create sync_records table (granular logs and retry data)
CREATE TABLE "sync_records" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "job_id" UUID,
  "integration_id" UUID NOT NULL,
  "source_record_unique_id" VARCHAR(255),
  "destination_record_id" VARCHAR(255),
  "status" VARCHAR(50) NOT NULL,
  "error_message" TEXT,
  "raw_data_payload" JSONB,
  "data_hash" VARCHAR(64),
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE
);

-- Create indexes for performance
CREATE INDEX "idx_users_email" ON "users" ("email");
CREATE INDEX "idx_users_deleted_at" ON "users" ("deleted_at");

CREATE INDEX "idx_connections_user_id" ON "connections" ("user_id");
CREATE INDEX "idx_connections_provider" ON "connections" ("provider");
CREATE INDEX "idx_connections_deleted_at" ON "connections" ("deleted_at");

CREATE INDEX "idx_integrations_user_id" ON "integrations" ("user_id");
CREATE INDEX "idx_integrations_source_connection_id" ON "integrations" ("source_connection_id");
CREATE INDEX "idx_integrations_destination_connection_id" ON "integrations" ("destination_connection_id");
CREATE INDEX "idx_integrations_status" ON "integrations" ("status");
CREATE INDEX "idx_integrations_deleted_at" ON "integrations" ("deleted_at");

CREATE INDEX "idx_sync_jobs_integration_id" ON "sync_jobs" ("integration_id");
CREATE INDEX "idx_sync_jobs_status" ON "sync_jobs" ("status");
CREATE INDEX "idx_sync_jobs_start_time" ON "sync_jobs" ("start_time");

CREATE INDEX "idx_sync_records_job_id" ON "sync_records" ("job_id");
CREATE INDEX "idx_sync_records_integration_id" ON "sync_records" ("integration_id");
CREATE INDEX "idx_sync_records_status" ON "sync_records" ("status");
CREATE INDEX "idx_sync_records_source_record_unique_id" ON "sync_records" ("source_record_unique_id");
CREATE INDEX "idx_sync_records_data_hash" ON "sync_records" ("data_hash");
CREATE INDEX "idx_sync_records_deleted_at" ON "sync_records" ("deleted_at");

-- Add foreign key constraints
ALTER TABLE "connections" ADD CONSTRAINT "fk_connections_user_id" 
  FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;

ALTER TABLE "integrations" ADD CONSTRAINT "fk_integrations_user_id" 
  FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;

ALTER TABLE "integrations" ADD CONSTRAINT "fk_integrations_source_connection_id" 
  FOREIGN KEY ("source_connection_id") REFERENCES "connections" ("id") ON DELETE SET NULL;

ALTER TABLE "integrations" ADD CONSTRAINT "fk_integrations_destination_connection_id" 
  FOREIGN KEY ("destination_connection_id") REFERENCES "connections" ("id") ON DELETE SET NULL;

ALTER TABLE "sync_jobs" ADD CONSTRAINT "fk_sync_jobs_integration_id" 
  FOREIGN KEY ("integration_id") REFERENCES "integrations" ("id") ON DELETE CASCADE;

ALTER TABLE "sync_records" ADD CONSTRAINT "fk_sync_records_job_id" 
  FOREIGN KEY ("job_id") REFERENCES "sync_jobs" ("id") ON DELETE CASCADE;

ALTER TABLE "sync_records" ADD CONSTRAINT "fk_sync_records_integration_id" 
  FOREIGN KEY ("integration_id") REFERENCES "integrations" ("id") ON DELETE CASCADE;





