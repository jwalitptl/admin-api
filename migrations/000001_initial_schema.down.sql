-- Drop constraints first
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_phone_format;
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_email_format;
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_type;
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_status;
ALTER TABLE organizations DROP CONSTRAINT IF EXISTS chk_organizations_status;
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS chk_accounts_status;

-- Drop indexes
DROP INDEX IF EXISTS idx_users_type;
DROP INDEX IF EXISTS idx_users_last_active_at;
DROP INDEX IF EXISTS idx_users_phone;
DROP INDEX IF EXISTS idx_users_name;
DROP INDEX IF EXISTS idx_users_status;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_organization_id;
DROP INDEX IF EXISTS idx_organizations_account_id;

-- Drop tables in correct order due to foreign key dependencies
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS accounts; 