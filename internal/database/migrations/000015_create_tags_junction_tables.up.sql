-- Connects tags to posts
CREATE TABLE post_tags (
    post_id     INTEGER,
    tag_id      INTEGER,
    PRIMARY KEY (tag_id, post_id),
    FOREIGN KEY (post_id) REFERENCES posts(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);

-- Connects tags to notes
CREATE TABLE note_tags (
    note_id     INTEGER,
    tag_id      INTEGER,
    PRIMARY KEY (tag_id, note_id),
    FOREIGN KEY (note_id) REFERENCES notes(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);

-- Connects tags to bookmarks
CREATE TABLE bookmark_tags (
    bookmark_id INTEGER,
    tag_id      INTEGER,
    PRIMARY KEY (tag_id, bookmark_id),
    FOREIGN KEY (bookmark_id) REFERENCES bookmarks(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);

-- Connects tags to projects
CREATE TABLE project_tags (
    project_id  INTEGER,
    tag_id      INTEGER,
    PRIMARY KEY (tag_id, project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);

-- Connects tags to open source contributions
CREATE TABLE opensource_contribution_tags (
    opensource_contribution_id INTEGER,
    tag_id                     INTEGER,
    PRIMARY KEY (tag_id, opensource_contribution_id),
    FOREIGN KEY (opensource_contribution_id) REFERENCES opensource_contributions(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);

-- TODO: Implement and integrate this query to the back end code,
--      SELECT a.* 
--      FROM articles a
--      JOIN article_tags at ON a.id = at.article_id
--      JOIN tags t ON at.tag_id = t.id
--      WHERE t.name = LOWER(?); 
