-- Rollback seed apps migration
DELETE FROM "apps" WHERE "name" IN ('googlesheet', 'salesforce', 'quickbooks', 'tally', 'googledrive');




