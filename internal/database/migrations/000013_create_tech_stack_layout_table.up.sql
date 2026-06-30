CREATE TABLE IF NOT EXISTS tech_stack_layouts (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    work_id             INTEGER NOT NULL UNIQUE,  -- UNIQUE ensures a 1-to-1 relationship
    tech_stack_json     TEXT NOT NULL DEFAULT '', -- Stores the '[["Go", "Typescript"...]]' string
    FOREIGN KEY (work_id) REFERENCES selected_works(id) ON DELETE CASCADE
);
