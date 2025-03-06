package user

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Repository interface {
	SignUp(ctx context.Context, arg signUpRequest) error
	SignIn(ctx context.Context, arg signInRequest) (signInResponse, error)
}

type repository struct {
	querier     *pgxpool.Pool
	redisClient *redis.Client
}

func NewRepository(querier *pgxpool.Pool, redisClient *redis.Client) Repository {
	return &repository{
		querier:     querier,
		redisClient: redisClient,
	}
}

func (r *repository) SignUp(ctx context.Context, arg signUpRequest) error {
	// Insert to database

	return nil
}

func (r *repository) SignIn(ctx context.Context, arg signInRequest) (signInResponse, error) {
	// TODO: Hash password
	// Insert to database

	session := session{
		SessionID: "foo",
		UserID:    "bar",
		ExpiresAt: time.Now().Add(time.Minute),
	}

	// Test
	err := r.redisClient.JSONSet(ctx, "session:id", "$", session).Err()
	if err != nil {
		return signInResponse{}, err
	}

	return signInResponse{
		Session: session,
	}, nil
}
