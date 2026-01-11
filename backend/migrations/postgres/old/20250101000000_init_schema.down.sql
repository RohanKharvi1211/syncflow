-- Drop foreign key constraints
ALTER TABLE "sync_records" DROP CONSTRAINT IF EXISTS "fk_sync_records_integration_id";
ALTER TABLE "sync_records" DROP CONSTRAINT IF EXISTS "fk_sync_records_job_id";
ALTER TABLE "sync_jobs" DROP CONSTRAINT IF EXISTS "fk_sync_jobs_integration_id";
ALTER TABLE "integrations" DROP CONSTRAINT IF EXISTS "fk_integrations_destination_connection_id";
ALTER TABLE "integrations" DROP CONSTRAINT IF EXISTS "fk_integrations_source_connection_id";
ALTER TABLE "integrations" DROP CONSTRAINT IF EXISTS "fk_integrations_user_id";
ALTER TABLE "connections" DROP CONSTRAINT IF EXISTS "fk_connections_user_id";

-- Drop indexes
DROP INDEX IF EXISTS "idx_sync_records_deleted_at";
DROP INDEX IF EXISTS "idx_sync_records_data_hash";
DROP INDEX IF EXISTS "idx_sync_records_source_record_unique_id";
DROP INDEX IF EXISTS "idx_sync_records_status";
DROP INDEX IF EXISTS "idx_sync_records_integration_id";
DROP INDEX IF EXISTS "idx_sync_records_job_id";
DROP INDEX IF EXISTS "idx_sync_jobs_start_time";
DROP INDEX IF EXISTS "idx_sync_jobs_status";
DROP INDEX IF EXISTS "idx_sync_jobs_integration_id";
DROP INDEX IF EXISTS "idx_integrations_deleted_at";
DROP INDEX IF EXISTS "idx_integrations_status";
DROP INDEX IF EXISTS "idx_integrations_destination_connection_id";
DROP INDEX IF EXISTS "idx_integrations_source_connection_id";
DROP INDEX IF EXISTS "idx_integrations_user_id";
DROP INDEX IF EXISTS "idx_connections_deleted_at";
DROP INDEX IF EXISTS "idx_connections_provider";
DROP INDEX IF EXISTS "idx_connections_user_id";
DROP INDEX IF EXISTS "idx_users_deleted_at";
DROP INDEX IF EXISTS "idx_users_email";

-- Drop tables
DROP TABLE IF EXISTS "sync_records";
DROP TABLE IF EXISTS "sync_jobs";
DROP TABLE IF EXISTS "integrations";
DROP TABLE IF EXISTS "connections";
DROP TABLE IF EXISTS "users";

-- Note: We don't drop the uuid-ossp extension as it might be used by other databases





