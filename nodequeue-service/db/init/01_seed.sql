-- Seed initial data (idempotent).
-- This file runs once when the Postgres data volume is first initialized.

INSERT INTO queue_service (id, notes)
VALUES (1, 'NodeQueueService singleton metadata row')
ON CONFLICT (id) DO NOTHING;

INSERT INTO resources (id, capacity)
VALUES
  ('Room 1', 5),
  ('Room 2', 3),
  ('Room 3', 4)
ON CONFLICT (id) DO NOTHING;


