-- Schema for NodeQueue persistence/audit layer.
-- This file runs once when the Postgres data volume is first initialized.

-- For UUID helpers (optional but handy for manual inserts).
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
  capacity   integer NOT NULL CHECK (capacity >= 0),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS nodes (
  id          uuid PRIMARY KEY,
  entity_id   uuid NOT NULL REFERENCES entities(id) ON DELETE RESTRICT,
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

CREATE INDEX IF NOT EXISTS idx_nodes_resource_id ON nodes(resource_id);
CREATE INDEX IF NOT EXISTS idx_node_logs_node_ts ON node_logs(node_id, ts);


