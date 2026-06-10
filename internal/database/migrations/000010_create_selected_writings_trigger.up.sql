CREATE TRIGGER keep_only_4_selected_writings
AFTER INSERT ON selected_writings
BEGIN
    DELETE FROM selected_writings
    WHERE id IN (
        SELECT id
        FROM selected_writings
        ORDER BY id ASC
        LIMIT (
            SELECT MAX(COUNT(*) - 4, 0)
            FROM selected_writings
        )
    );
END;
