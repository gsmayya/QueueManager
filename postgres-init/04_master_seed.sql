-- Seed data for master_db (idempotent).
-- Note: docker-entrypoint-initdb.d scripts run only on first volume init.

\connect master_db;

-- 10 sample entities (customers)
INSERT INTO entities (id, name, phone, email, joining_date)
VALUES
  ('11111111-1111-1111-1111-111111111111', 'Alice Johnson',  '5550000001', 'alice@example.com',  now() - interval '90 days'),
  ('22222222-2222-2222-2222-222222222222', 'Bob Smith',      '5550000002', 'bob@example.com',    now() - interval '80 days'),
  ('33333333-3333-3333-3333-333333333333', 'Carol Lee',      '5550000003', NULL,                 now() - interval '70 days'),
  ('44444444-4444-4444-4444-444444444444', 'David Kim',      '5550000004', 'david@example.com',  now() - interval '60 days'),
  ('55555555-5555-5555-5555-555555555555', 'Emma Davis',     '5550000005', NULL,                 now() - interval '50 days'),
  ('66666666-6666-6666-6666-666666666666', 'Frank Miller',   '5550000006', 'frank@example.com',  now() - interval '40 days'),
  ('77777777-7777-7777-7777-777777777777', 'Grace Wilson',   '5550000007', 'grace@example.com',  now() - interval '30 days'),
  ('88888888-8888-8888-8888-888888888888', 'Henry Brown',    '5550000008', NULL,                 now() - interval '20 days'),
  ('99999999-9999-9999-9999-999999999999', 'Ivy Martinez',   '5550000009', 'ivy@example.com',    now() - interval '10 days'),
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Jack Thompson',  '5550000010', 'jack@example.com',   now() - interval '5 days')
ON CONFLICT (id) DO NOTHING;

-- 10 sample users (system users)
-- password for all seeded users: "password123"
-- bcrypt hash generated via master-service module: $2a$10$E16dJIPZg8noTdupU3zRwOmvXc6aLyCG.Ueenl23knHuVkbFsKqHi
INSERT INTO users (id, user_id, name, email, is_admin, password_hash, created_at)
VALUES
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'admin',     'Admin User',     'admin@example.com',     true,  '$2a$10$E16dJIPZg8noTdupU3zRwOmvXc6aLyCG.Ueenl23knHuVkbFsKqHi', now() - interval '30 days'),
  ('cccccccc-cccc-cccc-cccc-cccccccccccc', 'user01',    'User 01',        'user01@example.com',    false, '$2a$10$E16dJIPZg8noTdupU3zRwOmvXc6aLyCG.Ueenl23knHuVkbFsKqHi', now() - interval '29 days'),
  ('dddddddd-dddd-dddd-dddd-dddddddddddd', 'user02',    'User 02',        'user02@example.com',    false, '$2a$10$E16dJIPZg8noTdupU3zRwOmvXc6aLyCG.Ueenl23knHuVkbFsKqHi', now() - interval '28 days'),
  ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', 'user03',    'User 03',        'user03@example.com',    false, '$2a$10$E16dJIPZg8noTdupU3zRwOmvXc6aLyCG.Ueenl23knHuVkbFsKqHi', now() - interval '27 days'),
  ('ffffffff-ffff-ffff-ffff-ffffffffffff', 'user04',    'User 04',        'user04@example.com',    false, '$2a$10$E16dJIPZg8noTdupU3zRwOmvXc6aLyCG.Ueenl23knHuVkbFsKqHi', now() - interval '26 days'),
  ('12121212-1212-1212-1212-121212121212', 'user05',    'User 05',        'user05@example.com',    false, '$2a$10$E16dJIPZg8noTdupU3zRwOmvXc6aLyCG.Ueenl23knHuVkbFsKqHi', now() - interval '25 days'),
  ('13131313-1313-1313-1313-131313131313', 'user06',    'User 06',        'user06@example.com',    false, '$2a$10$E16dJIPZg8noTdupU3zRwOmvXc6aLyCG.Ueenl23knHuVkbFsKqHi', now() - interval '24 days'),
  ('14141414-1414-1414-1414-141414141414', 'user07',    'User 07',        'user07@example.com',    false, '$2a$10$E16dJIPZg8noTdupU3zRwOmvXc6aLyCG.Ueenl23knHuVkbFsKqHi', now() - interval '23 days'),
  ('15151515-1515-1515-1515-151515151515', 'user08',    'User 08',        'user08@example.com',    false, '$2a$10$E16dJIPZg8noTdupU3zRwOmvXc6aLyCG.Ueenl23knHuVkbFsKqHi', now() - interval '22 days'),
  ('16161616-1616-1616-1616-161616161616', 'user09',    'User 09',        'user09@example.com',    false, '$2a$10$E16dJIPZg8noTdupU3zRwOmvXc6aLyCG.Ueenl23knHuVkbFsKqHi', now() - interval '21 days')
ON CONFLICT (id) DO NOTHING;

