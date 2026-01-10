-- Revert QuickBooks app back to destination only

UPDATE "apps" 
SET "type" = 'destination',
    "description" = 'Sync data to QuickBooks accounting software'
WHERE "name" = 'quickbooks';

