CREATE TRIGGER keep_only_4_selected_works
AFTER INSERT ON selected_works
BEGIN
    DELETE FROM selected_works
    WHERE id IN (
        SELECT id
        FROM selected_works
        ORDER BY id ASC
        LIMIT (
            SELECT MAX(COUNT(*) - 4, 0)
            FROM selected_works
        )
    );
END;
