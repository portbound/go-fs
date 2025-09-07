-- name: GetUser :one
SELECT * FROM users 
WHERE email = ? LIMIT 1;

-- name: CreateFileMeta :exec
INSERT INTO file_meta (
	id, parent_id, thumb_id, name, content_type, size, upload_date, owner
) VALUES (
	?, ?, ?, ?, ?, ?, ?, ?
);
	 
-- name: GetFileMeta :one
SELECT * FROM file_meta 
WHERE id = ? LIMIT 1;

-- name: GetAllFileMeta :many
SELECT * FROM file_meta
WHERE owner = ?
ORDER BY upload_date;

-- name: DeleteFileMeta :exec
DELETE FROM file_meta 
WHERE id = ?;
