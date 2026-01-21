-- Add `is_admin` to users and mark admin@example.com as admin.
-- This is idempotent and can be applied to existing databases.

\connect master_db;

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS is_admin boolean NOT NULL DEFAULT false;

UPDATE users
SET is_admin = true
WHERE email = 'admin@example.com';

