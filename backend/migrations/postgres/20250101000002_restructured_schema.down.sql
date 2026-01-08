-- Rollback for restructured schema migration

-- Drop foreign key constraints
ALTER TABLE "integrations" DROP CONSTRAINT IF EXISTS "fk_integrations_destination_connection_id";
ALTER TABLE "integrations" DROP CONSTRAINT IF EXISTS "fk_integrations_source_connection_id";
ALTER TABLE "integrations" DROP CONSTRAINT IF EXISTS "fk_integrations_destination_app_id";
ALTER TABLE "integrations" DROP CONSTRAINT IF EXISTS "fk_integrations_source_app_id";
ALTER TABLE "integrations" DROP CONSTRAINT IF EXISTS "fk_integrations_company_id";
ALTER TABLE "metadata" DROP CONSTRAINT IF EXISTS "fk_metadata_connection_id";
ALTER TABLE "connections" DROP CONSTRAINT IF EXISTS "fk_connections_app_id";
ALTER TABLE "connections" DROP CONSTRAINT IF EXISTS "fk_connections_company_id";
ALTER TABLE "users" DROP CONSTRAINT IF EXISTS "fk_users_company_id";

-- Drop indexes
DROP INDEX IF EXISTS "idx_integrations_deleted_at";
DROP INDEX IF EXISTS "idx_integrations_status";
DROP INDEX IF EXISTS "idx_integrations_destination_connection_id";
DROP INDEX IF EXISTS "idx_integrations_source_connection_id";
DROP INDEX IF EXISTS "idx_integrations_destination_app_id";
DROP INDEX IF EXISTS "idx_integrations_source_app_id";
DROP INDEX IF EXISTS "idx_integrations_company_id";
DROP INDEX IF EXISTS "idx_metadata_connection_id";
DROP INDEX IF EXISTS "idx_connections_deleted_at";
DROP INDEX IF EXISTS "idx_connections_is_active";
DROP INDEX IF EXISTS "idx_connections_app_id";
DROP INDEX IF EXISTS "idx_connections_company_id";
DROP INDEX IF EXISTS "idx_apps_deleted_at";
DROP INDEX IF EXISTS "idx_apps_is_active";
DROP INDEX IF EXISTS "idx_apps_type";
DROP INDEX IF EXISTS "idx_apps_name";
DROP INDEX IF EXISTS "idx_users_deleted_at";
DROP INDEX IF EXISTS "idx_users_invitation_token";
DROP INDEX IF EXISTS "idx_users_role";
DROP INDEX IF EXISTS "idx_users_email";
DROP INDEX IF EXISTS "idx_users_company_id";
DROP INDEX IF EXISTS "idx_companies_is_active";
DROP INDEX IF EXISTS "idx_companies_deleted_at";
DROP INDEX IF EXISTS "idx_companies_domain";

-- Drop tables (in reverse order of dependencies)
DROP TABLE IF EXISTS "integrations";
DROP TABLE IF EXISTS "metadata";
DROP TABLE IF EXISTS "connections";
DROP TABLE IF EXISTS "apps";
DROP TABLE IF EXISTS "users";
DROP TABLE IF EXISTS "companies";




