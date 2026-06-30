CREATE TABLE IF NOT EXISTS notes (
    id          INTEGER PRIMARY KEY,
    title       TEXT NOT NULL,
    slug        TEXT UNIQUE NOT NULL,
    content     TEXT NOT NULL DEFAULT '',
    tags        TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'draft',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
