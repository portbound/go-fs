-- name: Create :exec
INSERT INTO files (
	id, name, original_name, owner, type, size, unit, upload_date, storage_path
) VALUES (
	?, ?, ?, ?, ?, ?, ?, ?, ?
);
	 
-- name: Get :one
SELECT * FROM files 
WHERE id = ? LIMIT 1;

-- name: GetAll :many
SELECT * FROM files 
ORDER BY upload_date;

-- name: Update :exec
UPDATE files 
set name = ?, type = ?, size = ?, unit = ?, modified_date = ?, storage_path = ?
WHERE id = ?;

-- name: Delete :exec
DELETE FROM files 
WHERE id = ?;
