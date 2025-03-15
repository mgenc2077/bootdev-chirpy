-- name: UserByEmail :one
Select * from users WHERE email=$1;