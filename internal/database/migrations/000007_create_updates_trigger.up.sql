CREATE TRIGGER IF NOT EXISTS update_posts_updated_at
AFTER UPDATE ON posts
FOR EACH ROW
WHEN NEW.updated_at IS OLD.updated_at
  AND NEW.tags IS OLD.tags
  AND OLD.status = 'published'
  AND NEW.status = 'published'
BEGIN
    UPDATE posts SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_bookmarks_updated_at
AFTER UPDATE ON bookmarks
FOR EACH ROW
WHEN NEW.updated_at IS OLD.updated_at
  AND NEW.tags IS OLD.tags
  AND OLD.status = 'published'
  AND NEW.status = 'published'
BEGIN
    UPDATE bookmarks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_notes_updated_at
AFTER UPDATE ON notes
FOR EACH ROW
WHEN NEW.updated_at IS OLD.updated_at
  AND NEW.tags IS OLD.tags
  AND OLD.status = 'published'
  AND NEW.status = 'published'
BEGIN
    UPDATE notes SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_projects_updated_at
AFTER UPDATE ON projects
FOR EACH ROW
WHEN NEW.updated_at IS OLD.updated_at
  AND NEW.tags IS OLD.tags
  AND OLD.status = 'published'
  AND NEW.status = 'published'
BEGIN
    UPDATE projects SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_contributions_updated_at
AFTER UPDATE ON opensource_contributions
FOR EACH ROW
WHEN NEW.updated_at IS OLD.updated_at
  AND NEW.tags IS OLD.tags
BEGIN
    UPDATE opensource_contributions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
