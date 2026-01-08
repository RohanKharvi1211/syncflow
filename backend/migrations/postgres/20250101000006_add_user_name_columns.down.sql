-- Rollback: Remove added columns from users table

ALTER TABLE "users" DROP COLUMN IF EXISTS "invitation_accepted_at";
ALTER TABLE "users" DROP COLUMN IF EXISTS "invitation_sent_at";
DROP INDEX IF EXISTS "idx_users_invitation_token";
ALTER TABLE "users" DROP COLUMN IF EXISTS "invitation_token";
ALTER TABLE "users" DROP COLUMN IF EXISTS "is_active";
ALTER TABLE "users" DROP COLUMN IF EXISTS "role";
ALTER TABLE "users" DROP COLUMN IF EXISTS "last_name";
ALTER TABLE "users" DROP COLUMN IF EXISTS "first_name";


