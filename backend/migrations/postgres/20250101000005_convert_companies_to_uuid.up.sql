-- Convert companies table from INTEGER id to UUID
-- This migration handles the conversion from legacy INTEGER id to UUID

-- Step 1: Add a new UUID column
ALTER TABLE "companies" ADD COLUMN "id_new" UUID;

-- Step 2: Generate UUIDs for existing rows
UPDATE "companies" SET "id_new" = uuid_generate_v4();

-- Step 3: Update all foreign key references
-- Update users table
ALTER TABLE "users" ADD COLUMN "company_id_new" UUID;
UPDATE "users" u SET "company_id_new" = c."id_new" 
FROM "companies" c WHERE u."company_id"::text = c."id"::text;
ALTER TABLE "users" DROP CONSTRAINT IF EXISTS "fk_users_company_id";
ALTER TABLE "users" DROP COLUMN "company_id";
ALTER TABLE "users" RENAME COLUMN "company_id_new" TO "company_id";
ALTER TABLE "users" ADD CONSTRAINT "fk_users_company_id" 
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id_new") ON DELETE CASCADE;

-- Update connections table
ALTER TABLE "connections" ADD COLUMN "company_id_new" UUID;
UPDATE "connections" conn SET "company_id_new" = c."id_new" 
FROM "companies" c WHERE conn."company_id"::text = c."id"::text;
ALTER TABLE "connections" DROP CONSTRAINT IF EXISTS "fk_connections_company_id";
ALTER TABLE "connections" DROP COLUMN "company_id";
ALTER TABLE "connections" RENAME COLUMN "company_id_new" TO "company_id";
ALTER TABLE "connections" ADD CONSTRAINT "fk_connections_company_id" 
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id_new") ON DELETE CASCADE;

-- Update integrations table
ALTER TABLE "integrations" ADD COLUMN "company_id_new" UUID;
UPDATE "integrations" i SET "company_id_new" = c."id_new" 
FROM "companies" c WHERE i."company_id"::text = c."id"::text;
ALTER TABLE "integrations" DROP CONSTRAINT IF EXISTS "fk_integrations_company_id";
ALTER TABLE "integrations" DROP COLUMN "company_id";
ALTER TABLE "integrations" RENAME COLUMN "company_id_new" TO "company_id";
ALTER TABLE "integrations" ADD CONSTRAINT "fk_integrations_company_id" 
  FOREIGN KEY ("company_id") REFERENCES "companies" ("id_new") ON DELETE CASCADE;

-- Update pipelines table (if it exists)
DO $$ 
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'pipelines') THEN
    ALTER TABLE "pipelines" ADD COLUMN "company_id_new" UUID;
    EXECUTE 'UPDATE "pipelines" p SET "company_id_new" = c."id_new" 
             FROM "companies" c WHERE p."company_id"::text = c."id"::text';
    ALTER TABLE "pipelines" DROP CONSTRAINT IF EXISTS "fk_pipelines_company_id";
    ALTER TABLE "pipelines" DROP COLUMN "company_id";
    ALTER TABLE "pipelines" RENAME COLUMN "company_id_new" TO "company_id";
    ALTER TABLE "pipelines" ADD CONSTRAINT "fk_pipelines_company_id" 
      FOREIGN KEY ("company_id") REFERENCES "companies" ("id_new") ON DELETE CASCADE;
  END IF;
END $$;

-- Step 4: Drop old primary key and rename new column
ALTER TABLE "companies" DROP CONSTRAINT "companies_pkey";
ALTER TABLE "companies" DROP COLUMN "id";
ALTER TABLE "companies" RENAME COLUMN "id_new" TO "id";
ALTER TABLE "companies" ADD PRIMARY KEY ("id");
ALTER TABLE "companies" ALTER COLUMN "id" SET DEFAULT uuid_generate_v4();

-- Step 5: Update legacy tables that reference companies (if they exist)
-- These use INTEGER company_id, so we'll leave them as-is for now
-- They're legacy tables that will be deprecated


