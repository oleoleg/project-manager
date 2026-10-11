ALTER TABLE users
    ADD COLUMN IF NOT EXISTS theme TEXT NOT NULL DEFAULT 'light';

ALTER TABLE users
    ADD CONSTRAINT users_theme_check CHECK (theme IN ('light', 'dark'));