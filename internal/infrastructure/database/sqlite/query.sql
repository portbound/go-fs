-- name: Create :exec
INSERT INTO files (
	id, parent_id, thumb_id, name, content_type, size, upload_date, owner
) VALUES (
	?, ?, ?, ?, ?, ?, ?, ?
);
	 
-- name: Get :one
SELECT * FROM files 
WHERE id = ? LIMIT 1;

-- name: GetAll :many
SELECT * FROM files
ORDER BY upload_date;

-- name: Delete :exec
DELETE FROM files 
WHERE id = ?;
