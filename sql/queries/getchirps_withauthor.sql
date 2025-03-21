-- name: GetChirpsByAuthor :many
Select * from chirps where user_id=$1
ORDER BY created_at;