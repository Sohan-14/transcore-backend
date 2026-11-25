package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sohan-14/transcore-backend/internal/apperr"
	"github.com/sohan-14/transcore-backend/internal/modules/auth/internal/adapters/postgres/db"
)

const uniqueViolation = "23505"

// UserRepository persists users via sqlc-generated queries.
// ===============================================================
type UserRepository struct {
	q *db.Queries
}

// NewUserRepository Create UserRepository
// ===============================================================
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		q: db.New(pool),
	}
}

// Create inserts a user and returns the new ID.
// ===============================================================
func (r *UserRepository) Create(
	ctx context.Context,
	email, passwordHash string,
) (string, error) {
	row, err := r.q.CreateUser(
		ctx,
		db.CreateUserParams{
			Email:        email,
			PasswordHash: passwordHash,
		},
	)

	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == uniqueViolation {
			return "", apperr.New(apperr.KindConflict, "email_already_registered", "this email is already registered")
		}
		return "", apperr.Internal(err)
	}

	return row.ID, nil
}
