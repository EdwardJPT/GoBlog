CREATE TABLE IF NOT EXISTS projects (
    id                          INTEGER PRIMARY KEY,
    name                        TEXT NOT NULL,
    slug                        TEXT UNIQUE NOT NULL,
    description                 TEXT,
    home_page_description       TEXT,
    repo_link                   TEXT,
    demo_link                   TEXT,
    other_links                 TEXT,
    tags                        TEXT,
    status                      TEXT NOT NULL DEFAULT 'draft',
    created_at                  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at                  DATETIME DEFAULT CURRENT_TIMESTAMP
);
