-- name: GetUser :one
SELECT * FROM users 
WHERE email = ? LIMIT 1;

-- name: SaveMetadata :exec
INSERT INTO metadata (
	id, file_name, thumb_name, content_type, size, timestamp, user_id
) VALUES (
	?, ?, ?, ?, ?, ?, ?
);

-- name: UpdateMetadata :exec
UPDATE metadata
SET deleted_at = ?
WHERE id = ?;

-- name: GetMetadata :one
SELECT * FROM metadata 
WHERE id = ? 
AND user_id = ? LIMIT 1;

-- name: DeleteMetadata :exec
DELETE FROM metadata 
WHERE id = ?
AND user_id = ?;
