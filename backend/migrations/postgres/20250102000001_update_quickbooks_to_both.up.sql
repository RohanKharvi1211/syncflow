-- Update QuickBooks app to support both source and destination
-- This allows QuickBooks to be used as a data source (to download data) as well as a destination

UPDATE "apps" 
SET "type" = 'both',
    "description" = 'Sync data to/from QuickBooks accounting software'
WHERE "name" = 'quickbooks';

