CREATE TABLE IF NOT EXISTS selected_writings (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    target_table    TEXT NOT NULL,
    target_id       INTEGER NOT NULL,
    UNIQUE(target_table, target_id)
);
