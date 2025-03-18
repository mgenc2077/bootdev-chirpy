// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: changepassword.sql

package database

import (
	"context"

	"github.com/google/uuid"
)

const changePassword = `-- name: ChangePassword :one
UPDATE users
SET hashed_password=$1
WHERE id=$2
RETURNING id, created_at, updated_at, email, hashed_password
`

type ChangePasswordParams struct {
	HashedPassword string
	ID             uuid.UUID
}

func (q *Queries) ChangePassword(ctx context.Context, arg ChangePasswordParams) (User, error) {
	row := q.db.QueryRowContext(ctx, changePassword, arg.HashedPassword, arg.ID)
	var i User
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Email,
		&i.HashedPassword,
	)
	return i, err
}
