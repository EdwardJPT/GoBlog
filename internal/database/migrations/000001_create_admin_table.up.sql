CREATE TABLE IF NOT EXISTS admin (
    admin_id         INTEGER PRIMARY KEY,
    username         TEXT NOT NULL UNIQUE,
    password         TEXT NOT NULL,
    created_at       TEXT NOT NULL DEFAULT (DATETIME()),
    login_timestamp  TEXT NOT NULL DEFAULT ''
);
