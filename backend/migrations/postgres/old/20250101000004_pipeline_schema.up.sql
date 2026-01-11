-- Pipeline Schema Migration
-- This migration creates the new pipeline-based schema as per the design document

-- 1. Update connections table to add type, auth_type, and status columns
ALTER TABLE "connections" 
  ADD COLUMN IF NOT EXISTS "type" VARCHAR(50),
  ADD COLUMN IF NOT EXISTS "auth_type" VARCHAR(50) DEFAULT 'OAUTH2',
  ADD COLUMN IF NOT EXISTS "status" VARCHAR(50) DEFAULT 'ACTIVE';

-- 2. Data Objects table - represents what exactly to read/write
CREATE TABLE "data_objects" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "connection_id" UUID NOT NULL,
  "object_type" VARCHAR(50) NOT NULL, -- 'SHEET', 'TABLE', 'INVOICE', 'ENTITY'
  "identifier" TEXT NOT NULL,          -- sheet name, table name, object API name
  "config" JSONB DEFAULT '{}',         -- range, columns, filters
  "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  "deleted_at" TIMESTAMP WITH TIME ZONE,
  CONSTRAINT "fk_data_objects_connection_id" 
    FOREIGN KEY ("connection_id") REFERENCES "connections" ("id") ON DELETE CASCADE
);

-- 3. Pipelines table - source → destination mapping
CREATE TABLE "pipelines" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "company_id" UUID NOT NULL,
  "source_object_id" UUID NOT NULL,
  "destination_object_id" UUID NOT NULL,
  "sync_type" VARCHAR(50) DEFAULT 'PULL', -- 'PULL', 'PUSH', 'BIDIRECTIONAL'
  "schedule_interval" INTEGER DEFAULT 60,  -- minutes
  "status" VARCHAR(50) DEFAULT 'ACTIVE',   -- 'ACTIVE', 'PAUSED'
  "field_mapping" JSONB DEFAULT '{}',      -- field mapping configuration
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

-- 4. Checkpoints table - guarantees no data loss
CREATE TABLE "checkpoints" (
  "pipeline_id" UUID PRIMARY KEY,
  "last_sync_time" TIMESTAMP WITH TIME ZONE NOT NULL,
  "last_cursor" TEXT,                    -- row number, timestamp, CDC token
  "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  CONSTRAINT "fk_checkpoints_pipeline_id" 
    FOREIGN KEY ("pipeline_id") REFERENCES "pipelines" ("id") ON DELETE CASCADE
);

-- 5. Sync Runs table - observability for each pipeline execution
CREATE TABLE "sync_runs" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "pipeline_id" UUID NOT NULL,
  "status" VARCHAR(50) NOT NULL,         -- 'SUCCESS', 'FAILED', 'PARTIAL'
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


