-- Schema for NodeQueue persistence/audit layer.
-- Copied from queue-service/db/init/00_schema.sql so we can mount a single init directory.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS queue_service (
  id         integer PRIMARY KEY DEFAULT 1,
  created_at timestamptz NOT NULL DEFAULT now(),
  notes      text
);

CREATE TABLE IF NOT EXISTS entities (
  id         uuid PRIMARY KEY,
  name       text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS resources (
  id         text PRIMARY KEY,
  name       text NOT NULL,
  capacity   integer NOT NULL CHECK (capacity >= 0),
  deleted_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS nodes (
  id          uuid PRIMARY KEY,
  entity_id   uuid NOT NULL REFERENCES entities(id) ON DELETE RESTRICT,
  node_name   text,
  resource_id text REFERENCES resources(id) ON DELETE SET NULL,
  completed   boolean NOT NULL DEFAULT false,
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS node_logs (
  id          bigserial PRIMARY KEY,
  node_id     uuid NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  action      text NOT NULL,
  resource_id text REFERENCES resources(id) ON DELETE SET NULL,
  ts          timestamptz NOT NULL DEFAULT now(),
  details     jsonb
);

-- Scheduling: recurring creation of nodes with deadlines.
CREATE TABLE IF NOT EXISTS schedules (
  id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  entity_id          uuid NOT NULL REFERENCES entities(id) ON DELETE RESTRICT,
  resource_id        text NOT NULL REFERENCES resources(id) ON DELETE RESTRICT,
  interval_seconds   integer NOT NULL CHECK (interval_seconds > 0),
  time_limit_seconds integer NOT NULL CHECK (time_limit_seconds > 0),
  -- waiting_expiry_seconds controls how long a scheduled node may remain in WAITING before auto-expire.
  waiting_expiry_seconds integer NOT NULL CHECK (waiting_expiry_seconds > 0),
  -- ends_at stops future schedule firing after this timestamp (optional).
  ends_at           timestamptz,
  enabled            boolean NOT NULL DEFAULT true,
  next_run_at        timestamptz NOT NULL DEFAULT now(),
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now()
);

-- Extend nodes with schedule/deadline fields (idempotent).
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS schedule_id uuid;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS time_limit_seconds integer;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS waiting_expiry_seconds integer;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS assigned_at timestamptz;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS due_at timestamptz;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS expires_at timestamptz;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS delay_flag boolean NOT NULL DEFAULT false;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS expired boolean NOT NULL DEFAULT false;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS expired_at timestamptz;

DO $$
BEGIN
  -- Add FK only if it doesn't exist.
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'nodes_schedule_id_fkey'
  ) THEN
    ALTER TABLE nodes
      ADD CONSTRAINT nodes_schedule_id_fkey
      FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE SET NULL;
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_nodes_resource_id ON nodes(resource_id);
CREATE INDEX IF NOT EXISTS idx_node_logs_node_ts ON node_logs(node_id, ts);

CREATE INDEX IF NOT EXISTS idx_schedules_enabled_next_run_at ON schedules(enabled, next_run_at);
CREATE INDEX IF NOT EXISTS idx_nodes_schedule_active ON nodes(schedule_id, completed);
CREATE INDEX IF NOT EXISTS idx_nodes_delay_active ON nodes(delay_flag, completed);
CREATE INDEX IF NOT EXISTS idx_nodes_expiry_scan ON nodes(expired, completed, expires_at);
CREATE INDEX IF NOT EXISTS idx_nodes_schedule_expired ON nodes(schedule_id, expired, completed);

