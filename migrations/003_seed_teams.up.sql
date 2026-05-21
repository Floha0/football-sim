INSERT INTO teams (name, power) VALUES
    ('Chelsea', 85),
    ('Arsenal', 80),
    ('Manchester City', 90),
    ('Liverpool', 75)
ON CONFLICT (name) DO NOTHING;