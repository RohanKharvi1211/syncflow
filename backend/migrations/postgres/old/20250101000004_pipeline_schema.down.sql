-- Pipeline Schema Migration Rollback

-- Drop tables in reverse order (respecting foreign key constraints)
DROP TABLE IF EXISTS "sync_runs";
DROP TABLE IF EXISTS "checkpoints";
DROP TABLE IF EXISTS "pipelines";
DROP TABLE IF EXISTS "data_objects";

-- Remove columns from connections table
ALTER TABLE "connections" 
  DROP COLUMN IF EXISTS "type",
  DROP COLUMN IF EXISTS "auth_type",
  DROP COLUMN IF EXISTS "status";


