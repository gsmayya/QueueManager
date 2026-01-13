-- Create master_db if it doesn't exist.
-- Copied from queue-admin/db/init/00_create_master_db.sql so we can mount a single init directory.

SELECT 'CREATE DATABASE master_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'master_db')\gexec

