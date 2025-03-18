-- name: ChangePassword :one
UPDATE users
SET hashed_password=$1
WHERE id=$2
RETURNING *;