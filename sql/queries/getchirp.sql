-- name: GetChirp :one
Select * from chirps WHERE id=$1;