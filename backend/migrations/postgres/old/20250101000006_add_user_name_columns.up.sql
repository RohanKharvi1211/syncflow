-- Add missing first_name and last_name columns to users table
-- Also add role and is_active columns if they don't exist

-- Add first_name column
DO $$ 
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                 WHERE table_name = 'users' AND column_name = 'first_name') THEN
    ALTER TABLE "users" ADD COLUMN "first_name" VARCHAR(255);
  END IF;
END $$;

-- Add last_name column
DO $$ 
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                 WHERE table_name = 'users' AND column_name = 'last_name') THEN
    ALTER TABLE "users" ADD COLUMN "last_name" VARCHAR(255);
  END IF;
END $$;

-- Add role column
DO $$ 
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                 WHERE table_name = 'users' AND column_name = 'role') THEN
    ALTER TABLE "users" ADD COLUMN "role" VARCHAR(50) DEFAULT 'user';
  END IF;
END $$;

-- Add is_active column
DO $$ 
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                 WHERE table_name = 'users' AND column_name = 'is_active') THEN
    ALTER TABLE "users" ADD COLUMN "is_active" BOOLEAN DEFAULT TRUE;
  END IF;
END $$;

-- Add invitation_token column (optional, for future use)
DO $$ 
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                 WHERE table_name = 'users' AND column_name = 'invitation_token') THEN
    ALTER TABLE "users" ADD COLUMN "invitation_token" VARCHAR(255);
    CREATE UNIQUE INDEX IF NOT EXISTS "idx_users_invitation_token" ON "users" ("invitation_token") WHERE "invitation_token" IS NOT NULL;
  END IF;
END $$;

-- Add invitation_sent_at column
DO $$ 
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                 WHERE table_name = 'users' AND column_name = 'invitation_sent_at') THEN
    ALTER TABLE "users" ADD COLUMN "invitation_sent_at" TIMESTAMP WITH TIME ZONE;
  END IF;
END $$;

-- Add invitation_accepted_at column
DO $$ 
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                 WHERE table_name = 'users' AND column_name = 'invitation_accepted_at') THEN
    ALTER TABLE "users" ADD COLUMN "invitation_accepted_at" TIMESTAMP WITH TIME ZONE;
  END IF;
END $$;


