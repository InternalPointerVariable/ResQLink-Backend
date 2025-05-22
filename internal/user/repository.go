package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Repository interface {
	Get(ctx context.Context, userID string) (userResponse, error)
	SignUp(ctx context.Context, arg signUpRequest) error
	SignIn(ctx context.Context, arg signInRequest) (signInResponse, error)
	SignInAnonymous(ctx context.Context, anonID string) (signInAnonymousResponse, error)

	generateSessionToken() (string, error)
	createSession(ctx context.Context, token, userID string, isAnon bool) (session, error)
	validateSessionToken(ctx context.Context, token string) (sessionValidationResponse, error)
	invalidateSession(ctx context.Context, sessionID, userID string) error
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

func (r *repository) Get(ctx context.Context, userID string) (userResponse, error) {
	query := `
    SELECT 
        user_id, 
        created_at, 
        updated_at, 
        email, 
        first_name, 
        middle_name, 
        last_name,
        birth_date,
        role,
        EXTRACT(epoch FROM status_update_frequency)::INT AS status_update_frequency,
        is_location_shared
    FROM users
    WHERE user_id = ($1)
    `

	rows, err := r.querier.Query(ctx, query, userID)
	if err != nil {
		return userResponse{}, err
	}

	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[userResponse])
	if err != nil {
		return userResponse{}, err
	}

	return user, nil
}

func (r *repository) SignUp(ctx context.Context, arg signUpRequest) error {
	passwordHash, err := hashPassword(arg.Password)
	if err != nil {
		return err
	}

	query := `
    INSERT INTO users (
        email,
        password_hash, 
        first_name,
        middle_name,
        last_name,
        birth_date,
        role,
        status_update_frequency,
        is_location_shared
    )
    VALUES (
        $1, $2, $3, $4, $5, $6, $7,
        make_interval(mins => $8::int),
        $9
    )
    `

	if _, err := r.querier.Exec(ctx,
		query,
		arg.Email,
		passwordHash,
		arg.FirstName,
		arg.MiddleName,
		arg.LastName,
		arg.BirthDate,
		arg.Role,
		arg.StatusUpdateFrequency,
		arg.IsLocationShared,
	); err != nil {
		return err
	}

	return nil
}

var errInvalidPassword = errors.New("invalid password")

func (r *repository) SignIn(ctx context.Context, arg signInRequest) (signInResponse, error) {
	query := `SELECT password_hash FROM users WHERE email = ($1)`

	var hashedPassword string

	row := r.querier.QueryRow(ctx, query, arg.Email)
	if err := row.Scan(&hashedPassword); err != nil {
		return signInResponse{}, err
	}

	isMatch := checkPasswordHash(arg.Password, hashedPassword)
	if !isMatch {
		return signInResponse{}, errInvalidPassword
	}

	query = `
    SELECT 
        user_id, 
        created_at, 
        updated_at, 
        email, 
        first_name, 
        middle_name, 
        last_name,
        birth_date,
        role,
        EXTRACT(epoch FROM status_update_frequency)::INT AS status_update_frequency,
        is_location_shared
    FROM users
    WHERE email = ($1)
    `

	rows, err := r.querier.Query(ctx, query, arg.Email)
	if err != nil {
		return signInResponse{}, err
	}

	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[userResponse])
	if err != nil {
		return signInResponse{}, err
	}

	token, err := r.generateSessionToken()
	if err != nil {
		return signInResponse{}, err
	}

	_, err = r.createSession(ctx, token, user.UserID, false)
	if err != nil {
		return signInResponse{}, err
	}

	return signInResponse{
		User:  user,
		Token: token,
	}, nil
}

type signInAnonymousRequest struct {
	AnonymousID string `json:"anonymousId"`
}

type signInAnonymousResponse struct {
	Token string `json:"token"`
}

func (r *repository) SignInAnonymous(
	ctx context.Context,
	anonID string,
) (signInAnonymousResponse, error) {
	token, err := r.generateSessionToken()
	if err != nil {
		return signInAnonymousResponse{}, err
	}

	_, err = r.createSession(ctx, token, anonID, true)
	if err != nil {
		return signInAnonymousResponse{}, err
	}

	return signInAnonymousResponse{
		Token: token,
	}, nil
}
