-- Remove unique constraint on company_id + app_id to allow multiple connections per app
-- This enables creating separate connection entries for each sheet/object
ALTER TABLE "connections" DROP CONSTRAINT IF EXISTS "unique_company_app_connection";


