-- Drop foreign key constraints for legacy tables
ALTER TABLE "tally_configs" DROP CONSTRAINT IF EXISTS "fk_tally_configs_company_id";
ALTER TABLE "sync_logs" DROP CONSTRAINT IF EXISTS "fk_sync_logs_sync_record_id";
ALTER TABLE "sync_logs" DROP CONSTRAINT IF EXISTS "fk_sync_logs_company_id";
ALTER TABLE "legacy_sync_records" DROP CONSTRAINT IF EXISTS "fk_legacy_sync_records_company_id";
ALTER TABLE "field_mappings" DROP CONSTRAINT IF EXISTS "fk_field_mappings_company_id";
ALTER TABLE "legacy_integrations" DROP CONSTRAINT IF EXISTS "fk_legacy_integrations_company_id";

-- Drop indexes for legacy tables
DROP INDEX IF EXISTS "idx_tally_configs_company_id";
DROP INDEX IF EXISTS "idx_sync_logs_sync_record_id";
DROP INDEX IF EXISTS "idx_sync_logs_company_id";
DROP INDEX IF EXISTS "idx_legacy_sync_records_deleted_at";
DROP INDEX IF EXISTS "idx_legacy_sync_records_status";
DROP INDEX IF EXISTS "idx_legacy_sync_records_company_id";
DROP INDEX IF EXISTS "idx_field_mappings_source_target";
DROP INDEX IF EXISTS "idx_field_mappings_company_id";
DROP INDEX IF EXISTS "idx_legacy_integrations_type";
DROP INDEX IF EXISTS "idx_legacy_integrations_company_id";
DROP INDEX IF EXISTS "idx_companies_deleted_at";
DROP INDEX IF EXISTS "idx_companies_domain";

-- Drop legacy tables
DROP TABLE IF EXISTS "tally_configs";
DROP TABLE IF EXISTS "sync_logs";
DROP TABLE IF EXISTS "legacy_sync_records";
DROP TABLE IF EXISTS "field_mappings";
DROP TABLE IF EXISTS "legacy_integrations";
DROP TABLE IF EXISTS "companies";





