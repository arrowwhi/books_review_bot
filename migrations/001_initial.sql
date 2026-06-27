-- +goose Up

CREATE TABLE IF NOT EXISTS genres (
    id         SERIAL PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE
);

INSERT INTO genres (name, is_default) VALUES
    ('Фантастика', TRUE),
    ('Детектив', TRUE),
    ('Историческая', TRUE),
    ('Нон-фикшн', TRUE)
ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS books (
    id             SERIAL PRIMARY KEY,
    user_id        BIGINT NOT NULL,
    title          TEXT NOT NULL,
    author         TEXT,
    genre_id       INTEGER REFERENCES genres(id) ON DELETE SET NULL,
    ol_key         TEXT,
    cover_url      TEXT,
    status         TEXT NOT NULL CHECK (status IN ('read', 'wishlist')),
    rating         SMALLINT CHECK (rating BETWEEN 1 AND 5),
    emotion        TEXT CHECK (emotion IN ('love','like','neutral','dislike','mixed')),
    aspect_plot    SMALLINT CHECK (aspect_plot BETWEEN 1 AND 10),
    aspect_chars   SMALLINT CHECK (aspect_chars BETWEEN 1 AND 10),
    aspect_atmo    SMALLINT CHECK (aspect_atmo BETWEEN 1 AND 10),
    aspect_ideas   SMALLINT CHECK (aspect_ideas BETWEEN 1 AND 10),
    aspect_style   SMALLINT CHECK (aspect_style BETWEEN 1 AND 10),
    aspect_tempo   SMALLINT CHECK (aspect_tempo BETWEEN 1 AND 10),
    liked_text     TEXT,
    disliked_text  TEXT,
    insight_text   TEXT,
    recommend      BOOLEAN,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at    TIMESTAMPTZ
);

CREATE INDEX idx_books_user_id     ON books(user_id);
CREATE INDEX idx_books_user_status ON books(user_id, status);

CREATE TABLE IF NOT EXISTS reminders (
    user_id        BIGINT PRIMARY KEY,
    interval_days  INTEGER NOT NULL DEFAULT 14,
    last_sent_at   TIMESTAMPTZ,
    enabled        BOOLEAN NOT NULL DEFAULT TRUE
);

-- +goose Down
DROP TABLE IF EXISTS reminders;
DROP TABLE IF EXISTS books;
DROP TABLE IF EXISTS genres;
