-- Schema for master_db.
-- Copied from queue-admin/db/init/01_schema.sql so we can mount a single init directory.

\connect master_db;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS entities (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name         text NOT NULL,
  phone        text NOT NULL,
  email        text,
  joining_date timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT entities_name_phone_uniq UNIQUE (name, phone)
);

CREATE TABLE IF NOT EXISTS users (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       text NOT NULL UNIQUE,
  name          text NOT NULL,
  email         text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_entities_joining_date ON entities(joining_date);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);

