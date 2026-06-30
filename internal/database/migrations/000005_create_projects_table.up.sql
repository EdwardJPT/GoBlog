CREATE TABLE IF NOT EXISTS projects (
    id                          INTEGER PRIMARY KEY,
    name                        TEXT NOT NULL,
    slug                        TEXT UNIQUE NOT NULL,
    description                 TEXT NOT NULL DEFAULT '',
    home_page_description       TEXT NOT NULL DEFAULT '',
    repo_link                   TEXT NOT NULL DEFAULT '',
    demo_link                   TEXT NOT NULL DEFAULT '',
    other_links                 TEXT NOT NULL DEFAULT '',
    tags                        TEXT NOT NULL DEFAULT '',
    status                      TEXT NOT NULL DEFAULT 'draft',
    created_at                  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at                  DATETIME DEFAULT CURRENT_TIMESTAMP
);
