-- Create master_db if it doesn't exist.
-- Note: This runs only when the Postgres data directory is first initialized.

SELECT 'CREATE DATABASE master_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'master_db')\gexec

