-- Re-add unique constraint on company_id + app_id
ALTER TABLE "connections" ADD CONSTRAINT "unique_company_app_connection" UNIQUE ("company_id", "app_id");


