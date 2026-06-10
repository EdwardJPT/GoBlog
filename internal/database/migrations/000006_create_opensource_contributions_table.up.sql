CREATE TABLE IF NOT EXISTS opensource_contributions (
    id                          INTEGER PRIMARY KEY,
    project_name                TEXT NOT NULL,
    slug                        TEXT UNIQUE NOT NULL,
    description                 TEXT,
    home_page_description       TEXT,
    tags                        TEXT,
    link                        TEXT,
    contribution_summary        TEXT,
    created_at                  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at                  DATETIME DEFAULT CURRENT_TIMESTAMP
);
