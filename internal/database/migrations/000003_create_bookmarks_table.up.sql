CREATE TABLE IF NOT EXISTS bookmarks (
    id          INTEGER PRIMARY KEY,
    title       TEXT NOT NULL,
    slug        TEXT UNIQUE NOT NULL,
    url         TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    tags        TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'draft',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
