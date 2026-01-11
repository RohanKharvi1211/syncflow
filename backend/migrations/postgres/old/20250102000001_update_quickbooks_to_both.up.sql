-- Update QuickBooks app to support both source and destination
-- This migration updates the QuickBooks app type to 'both'
-- This migration is idempotent - it checks if the apps table exists first

DO $$
BEGIN
    -- Check if apps table exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'apps'
    ) THEN
        RAISE NOTICE 'Apps table does not exist. Skipping QuickBooks app update.';
        RETURN;
    END IF;
    
    -- Update QuickBooks app type to 'both' if it exists
    IF EXISTS (
        SELECT 1 FROM "apps" WHERE "name" = 'quickbooks'
    ) THEN
        UPDATE "apps"
        SET "type" = 'both',
            "description" = 'Sync data to/from QuickBooks accounting software'
        WHERE "name" = 'quickbooks' AND "type" <> 'both';
        RAISE NOTICE 'QuickBooks app updated to type ''both''.';
    ELSE
        RAISE NOTICE 'QuickBooks app not found. Skipping update.';
    END IF;
END $$;
