-- Seed initial data (idempotent).
-- Copied from nodequeue-service/db/init/01_seed.sql so we can mount a single init directory.

INSERT INTO queue_service (id, notes)
VALUES (1, 'NodeQueueService singleton metadata row')
ON CONFLICT (id) DO NOTHING;

INSERT INTO resources (id, name, capacity)
VALUES
  ('room-1', 'Room 1', 5),
  ('room-2', 'Room 2', 3),
  ('room-3', 'Room 3', 4)
ON CONFLICT (id) DO NOTHING;

