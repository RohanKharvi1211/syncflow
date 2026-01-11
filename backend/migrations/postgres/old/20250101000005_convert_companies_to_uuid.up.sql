-- Convert companies table from INTEGER id to UUID
-- This migration handles the conversion from legacy INTEGER id to UUID
-- This migration is idempotent - it checks current state before making changes

-- Check if companies.id is already UUID
DO $$
DECLARE
    current_type text;
BEGIN
    SELECT data_type INTO current_type
    FROM information_schema.columns
    WHERE table_name = 'companies' AND column_name = 'id';
    
    IF current_type = 'uuid' THEN
        RAISE NOTICE 'Companies.id is already UUID. Skipping conversion.';
        RETURN;
    END IF;
    
    RAISE NOTICE 'Companies.id is %. Converting to UUID...', current_type;
END $$;

-- Step 1: Add a new UUID column (if it doesn't exist)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'companies' AND column_name = 'id_new'
    ) THEN
        ALTER TABLE "companies" ADD COLUMN "id_new" UUID;
    END IF;
END $$;

-- Step 2: Generate UUIDs for existing rows (only if id_new is NULL)
UPDATE "companies" SET "id_new" = uuid_generate_v4() WHERE "id_new" IS NULL;

-- Step 3: Update all foreign key references
-- Update users table (only if company_id is not already UUID)
DO $$
DECLARE
    users_company_id_type text;
    column_exists boolean;
BEGIN
    -- Check if column exists
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'users' AND column_name = 'company_id'
    ) INTO column_exists;
    
    IF NOT column_exists THEN
        RAISE NOTICE 'Users.company_id column does not exist. Skipping conversion.';
        RETURN;
    END IF;
    
    SELECT data_type INTO users_company_id_type
    FROM information_schema.columns
    WHERE table_name = 'users' AND column_name = 'company_id';
    
    IF users_company_id_type = 'uuid' THEN
        RAISE NOTICE 'Users.company_id is already UUID. Skipping conversion.';
        RETURN;
    END IF;
    
    IF users_company_id_type != 'uuid' THEN
        -- Add new column if it doesn't exist
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'users' AND column_name = 'company_id_new'
        ) THEN
            ALTER TABLE "users" ADD COLUMN "company_id_new" UUID;
        END IF;
        
        -- Update values
        UPDATE "users" u SET "company_id_new" = c."id_new" 
        FROM "companies" c WHERE u."company_id"::text = c."id"::text;
        
        -- Drop old column and rename
        ALTER TABLE "users" DROP CONSTRAINT IF EXISTS "fk_users_company_id";
        ALTER TABLE "users" DROP COLUMN IF EXISTS "company_id";
        ALTER TABLE "users" RENAME COLUMN "company_id_new" TO "company_id";
        ALTER TABLE "users" ADD CONSTRAINT "fk_users_company_id" 
          FOREIGN KEY ("company_id") REFERENCES "companies" ("id_new") ON DELETE CASCADE;
    ELSE
        RAISE NOTICE 'Users.company_id is already UUID. Skipping conversion.';
    END IF;
END $$;

-- Update connections table (only if company_id is not already UUID)
DO $$
DECLARE
    conn_company_id_type text;
    column_exists boolean;
BEGIN
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'connections' AND column_name = 'company_id'
    ) INTO column_exists;
    
    IF NOT column_exists THEN
        RAISE NOTICE 'Connections.company_id column does not exist. Skipping conversion.';
        RETURN;
    END IF;
    
    SELECT data_type INTO conn_company_id_type
    FROM information_schema.columns
    WHERE table_name = 'connections' AND column_name = 'company_id';
    
    IF conn_company_id_type = 'uuid' THEN
        RAISE NOTICE 'Connections.company_id is already UUID. Skipping conversion.';
        RETURN;
    END IF;
    
    IF conn_company_id_type != 'uuid' THEN
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'connections' AND column_name = 'company_id_new'
        ) THEN
            ALTER TABLE "connections" ADD COLUMN "company_id_new" UUID;
        END IF;
        
        UPDATE "connections" conn SET "company_id_new" = c."id_new" 
        FROM "companies" c WHERE conn."company_id"::text = c."id"::text;
        
        ALTER TABLE "connections" DROP CONSTRAINT IF EXISTS "fk_connections_company_id";
        ALTER TABLE "connections" DROP COLUMN IF EXISTS "company_id";
        ALTER TABLE "connections" RENAME COLUMN "company_id_new" TO "company_id";
        ALTER TABLE "connections" ADD CONSTRAINT "fk_connections_company_id" 
          FOREIGN KEY ("company_id") REFERENCES "companies" ("id_new") ON DELETE CASCADE;
    ELSE
        RAISE NOTICE 'Connections.company_id is already UUID. Skipping conversion.';
    END IF;
END $$;

-- Update integrations table (only if company_id is not already UUID)
DO $$
DECLARE
    int_company_id_type text;
    column_exists boolean;
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'integrations') THEN
        SELECT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = 'integrations' AND column_name = 'company_id'
        ) INTO column_exists;
        
        IF NOT column_exists THEN
            RAISE NOTICE 'Integrations.company_id column does not exist. Skipping conversion.';
            RETURN;
        END IF;
        
        SELECT data_type INTO int_company_id_type
        FROM information_schema.columns
        WHERE table_name = 'integrations' AND column_name = 'company_id';
        
        IF int_company_id_type = 'uuid' THEN
            RAISE NOTICE 'Integrations.company_id is already UUID. Skipping conversion.';
            RETURN;
        END IF;
        
        IF int_company_id_type != 'uuid' THEN
            IF NOT EXISTS (
                SELECT 1 FROM information_schema.columns 
                WHERE table_name = 'integrations' AND column_name = 'company_id_new'
            ) THEN
                ALTER TABLE "integrations" ADD COLUMN "company_id_new" UUID;
            END IF;
            
            UPDATE "integrations" i SET "company_id_new" = c."id_new" 
            FROM "companies" c WHERE i."company_id"::text = c."id"::text;
            
            ALTER TABLE "integrations" DROP CONSTRAINT IF EXISTS "fk_integrations_company_id";
            ALTER TABLE "integrations" DROP COLUMN IF EXISTS "company_id";
            ALTER TABLE "integrations" RENAME COLUMN "company_id_new" TO "company_id";
            ALTER TABLE "integrations" ADD CONSTRAINT "fk_integrations_company_id" 
              FOREIGN KEY ("company_id") REFERENCES "companies" ("id_new") ON DELETE CASCADE;
        ELSE
            RAISE NOTICE 'Integrations.company_id is already UUID. Skipping conversion.';
        END IF;
    END IF;
END $$;

-- Update pipelines table (if it exists, only if company_id is not already UUID)
DO $$ 
DECLARE
    pipe_company_id_type text;
    column_exists boolean;
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'pipelines') THEN
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'pipelines' AND column_name = 'company_id'
    ) INTO column_exists;
    
    IF NOT column_exists THEN
        RAISE NOTICE 'Pipelines.company_id column does not exist. Skipping conversion.';
        RETURN;
    END IF;
    
    SELECT data_type INTO pipe_company_id_type
    FROM information_schema.columns
    WHERE table_name = 'pipelines' AND column_name = 'company_id';
    
    IF pipe_company_id_type = 'uuid' THEN
        RAISE NOTICE 'Pipelines.company_id is already UUID. Skipping conversion.';
        RETURN;
    END IF;
    
    IF pipe_company_id_type != 'uuid' THEN
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'pipelines' AND column_name = 'company_id_new'
        ) THEN
            ALTER TABLE "pipelines" ADD COLUMN "company_id_new" UUID;
        END IF;
        
        EXECUTE 'UPDATE "pipelines" p SET "company_id_new" = c."id_new" 
                 FROM "companies" c WHERE p."company_id"::text = c."id"::text';
        
        ALTER TABLE "pipelines" DROP CONSTRAINT IF EXISTS "fk_pipelines_company_id";
        ALTER TABLE "pipelines" DROP COLUMN IF EXISTS "company_id";
        ALTER TABLE "pipelines" RENAME COLUMN "company_id_new" TO "company_id";
        ALTER TABLE "pipelines" ADD CONSTRAINT "fk_pipelines_company_id" 
          FOREIGN KEY ("company_id") REFERENCES "companies" ("id_new") ON DELETE CASCADE;
    ELSE
        RAISE NOTICE 'Pipelines.company_id is already UUID. Skipping conversion.';
    END IF;
  END IF;
END $$;

-- Step 4: Drop old primary key and rename new column (only if id is not already UUID)
DO $$
DECLARE
    current_type text;
BEGIN
    SELECT data_type INTO current_type
    FROM information_schema.columns
    WHERE table_name = 'companies' AND column_name = 'id';
    
    IF current_type != 'uuid' AND EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'companies' AND column_name = 'id_new'
    ) THEN
        ALTER TABLE "companies" DROP CONSTRAINT IF EXISTS "companies_pkey";
        ALTER TABLE "companies" DROP COLUMN IF EXISTS "id";
        ALTER TABLE "companies" RENAME COLUMN "id_new" TO "id";
        ALTER TABLE "companies" ADD PRIMARY KEY ("id");
        ALTER TABLE "companies" ALTER COLUMN "id" SET DEFAULT uuid_generate_v4();
        RAISE NOTICE 'Companies.id converted to UUID successfully.';
    ELSIF current_type = 'uuid' THEN
        -- Clean up id_new if it exists (leftover from partial migration)
        IF EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'companies' AND column_name = 'id_new'
        ) THEN
            ALTER TABLE "companies" DROP COLUMN IF EXISTS "id_new";
        END IF;
        RAISE NOTICE 'Companies.id is already UUID. No conversion needed.';
    END IF;
END $$;

-- Step 5: Update legacy tables that reference companies (if they exist)
-- These use INTEGER company_id, so we'll leave them as-is for now
-- They're legacy tables that will be deprecated


