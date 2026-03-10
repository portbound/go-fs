-- name: GetUser :one
SELECT * FROM users 
WHERE email = ? LIMIT 1;

-- name: SaveMetadata :exec
INSERT INTO metadata (
	id, file_name, thumb_name, user_id
) VALUES (
	?, ?, ?, ?
);

-- name: GetMetadata :one
SELECT * FROM metadata 
WHERE id = ? 
AND user_id = ? LIMIT 1;

-- name: GetAllMetadata :many
SELECT * FROM metadata 
WHERE user_id = ?;

-- name: DeleteMetadata :exec
DELETE FROM metadata 
WHERE id = ?
AND user_id = ?;
