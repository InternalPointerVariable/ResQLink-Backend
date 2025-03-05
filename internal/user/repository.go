package user

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	SignUp(ctx context.Context, arg signUpRequest) error
	SignIn(ctx context.Context, arg signInRequest) (signInResponse, error)
}

type repository struct {
	querier *pgxpool.Pool
}

func NewRepository(querier *pgxpool.Pool) Repository {
	return &repository{
		querier: querier,
	}
}

func (r *repository) SignUp(ctx context.Context, arg signUpRequest) error {
    // Insert to database

	return nil
}

func (r *repository) SignIn(ctx context.Context, arg signInRequest) (signInResponse, error) {
    // TODO: Hash password
    // Insert to database

	return signInResponse{}, nil
}
