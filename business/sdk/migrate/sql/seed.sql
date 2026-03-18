-- Seed contexts
INSERT INTO contexts (context_id, title, description, status, summary) VALUES
    ('a1000000-0000-0000-0000-000000000001', 'Bathroom renovation', 'Full remodel of the master bathroom', 'active', 'Permits submitted. Waiting on contractor to start demo.'),
    ('a1000000-0000-0000-0000-000000000002', '2024 Taxes', 'Federal and state tax filing', 'active', 'Need to gather W2 and 1099 forms. Deadline April 15.'),
    ('a1000000-0000-0000-0000-000000000003', 'Career development', 'Ongoing professional growth', 'active', 'Working through Go backend architecture patterns.');

-- Seed tasks
INSERT INTO tasks (task_id, context_id, title, description, status, priority, energy, duration_min, due_date) VALUES
    ('b1000000-0000-0000-0000-000000000001', 'a1000000-0000-0000-0000-000000000001', 'Get contractor quotes', 'Need at least 3 quotes for comparison', 'in_progress', 'high', 'medium', 120, NULL),
    ('b1000000-0000-0000-0000-000000000002', 'a1000000-0000-0000-0000-000000000001', 'Select tile for shower', 'Visit tile showroom', 'todo', 'medium', 'low', 90, NULL),
    ('b1000000-0000-0000-0000-000000000003', 'a1000000-0000-0000-0000-000000000002', 'Gather W2 forms', 'Download from employer portal', 'todo', 'high', 'low', 30, '2025-03-15T00:00:00Z'),
    ('b1000000-0000-0000-0000-000000000004', 'a1000000-0000-0000-0000-000000000002', 'Schedule CPA appointment', '', 'todo', 'medium', 'low', 15, '2025-03-01T00:00:00Z'),
    ('b1000000-0000-0000-0000-000000000005', NULL, 'Call dentist', 'Schedule cleaning appointment', 'todo', 'low', 'low', 10, '2025-02-28T00:00:00Z'),
    ('b1000000-0000-0000-0000-000000000006', 'a1000000-0000-0000-0000-000000000003', 'Read Ardan Labs service repo', 'Study the architecture patterns', 'done', 'medium', 'high', 180, NULL);

-- Seed tags
INSERT INTO tags (tag_id, name) VALUES
    ('c1000000-0000-0000-0000-000000000001', 'home'),
    ('c1000000-0000-0000-0000-000000000002', 'finance'),
    ('c1000000-0000-0000-0000-000000000003', 'health'),
    ('c1000000-0000-0000-0000-000000000004', 'career');

-- Seed tag associations
INSERT INTO task_tags (task_id, tag_id) VALUES
    ('b1000000-0000-0000-0000-000000000001', 'c1000000-0000-0000-0000-000000000001'),
    ('b1000000-0000-0000-0000-000000000002', 'c1000000-0000-0000-0000-000000000001'),
    ('b1000000-0000-0000-0000-000000000003', 'c1000000-0000-0000-0000-000000000002'),
    ('b1000000-0000-0000-0000-000000000005', 'c1000000-0000-0000-0000-000000000003');

INSERT INTO context_tags (context_id, tag_id) VALUES
    ('a1000000-0000-0000-0000-000000000001', 'c1000000-0000-0000-0000-000000000001'),
    ('a1000000-0000-0000-0000-000000000002', 'c1000000-0000-0000-0000-000000000002'),
    ('a1000000-0000-0000-0000-000000000003', 'c1000000-0000-0000-0000-000000000004');

-- Seed context events
INSERT INTO context_events (event_id, context_id, kind, content) VALUES
    ('d1000000-0000-0000-0000-000000000001', 'a1000000-0000-0000-0000-000000000001', 'note', 'Started researching contractors in the area'),
    ('d1000000-0000-0000-0000-000000000002', 'a1000000-0000-0000-0000-000000000001', 'note', 'Permit application submitted to city'),
    ('d1000000-0000-0000-0000-000000000003', 'a1000000-0000-0000-0000-000000000002', 'note', 'Tax year started, need to begin collecting documents');
