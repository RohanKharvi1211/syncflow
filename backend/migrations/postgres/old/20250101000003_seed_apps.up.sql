-- Seed initial apps with their metadata schemas
-- This migration populates the apps table with common applications
-- This migration is idempotent - it checks if the apps table exists first

DO $$
BEGIN
    -- Check if apps table exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'apps'
    ) THEN
        RAISE NOTICE 'Apps table does not exist. Skipping app seeding.';
        RETURN;
    END IF;
END $$;

-- Google Sheets App (source)
INSERT INTO "apps" ("id", "name", "display_name", "description", "type", "metadata_schema", "is_active") 
SELECT
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
WHERE NOT EXISTS (SELECT 1 FROM "apps" WHERE "name" = 'googlesheet');

-- Salesforce App (source and destination)
INSERT INTO "apps" ("id", "name", "display_name", "description", "type", "metadata_schema", "is_active") 
SELECT
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
WHERE NOT EXISTS (SELECT 1 FROM "apps" WHERE "name" = 'salesforce');

-- QuickBooks App (source and destination)
INSERT INTO "apps" ("id", "name", "display_name", "description", "type", "metadata_schema", "is_active") 
SELECT
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
WHERE NOT EXISTS (SELECT 1 FROM "apps" WHERE "name" = 'quickbooks');

-- Tally App (destination)
INSERT INTO "apps" ("id", "name", "display_name", "description", "type", "metadata_schema", "is_active") 
SELECT
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
WHERE NOT EXISTS (SELECT 1 FROM "apps" WHERE "name" = 'tally');

-- Google Drive App (source)
INSERT INTO "apps" ("id", "name", "display_name", "description", "type", "metadata_schema", "is_active") 
SELECT
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
WHERE NOT EXISTS (SELECT 1 FROM "apps" WHERE "name" = 'googledrive');




