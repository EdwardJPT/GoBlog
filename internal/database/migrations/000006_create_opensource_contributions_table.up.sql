CREATE TABLE IF NOT EXISTS opensource_contributions (
    id                          INTEGER PRIMARY KEY,
    project_name                TEXT NOT NULL,
    slug                        TEXT UNIQUE NOT NULL,
    description                 TEXT NOT NULL DEFAULT '',
    home_page_description       TEXT NOT NULL DEFAULT '',
    tags                        TEXT NOT NULL DEFAULT '',
    link                        TEXT NOT NULL DEFAULT '',
    contribution_summary        TEXT NOT NULL DEFAULT '',
    created_at                  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at                  DATETIME DEFAULT CURRENT_TIMESTAMP
);
