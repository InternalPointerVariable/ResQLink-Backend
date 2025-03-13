package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Repository interface {
	SignUp(ctx context.Context, arg signUpRequest) error
	SignIn(ctx context.Context, arg signInRequest) (signInResponse, error)
	SaveLocation(ctx context.Context, arg saveLocationRequest) error
	GetLocation(ctx context.Context, userID string) (location, error)

	generateSessionToken() (string, error)
	createSession(ctx context.Context, token, userID string) (session, error)
	validateSessionToken(ctx context.Context, token string) (session, error)
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

	ses, err := r.createSession(ctx, token, user.UserID)
	if err != nil {
		return signInResponse{}, err
	}

	return signInResponse{
		Session: ses,
		User:    user,
	}, nil
}

type saveLocationRequest struct {
	UserID    string  `json:"userId"`
	Longitude int     `json:"longitude"`
	Latitude  int     `json:"latitude"`
	Address   *string `json:"address"`
}

type location struct {
	Longitude int     `json:"longitude"`
	Latitude  int     `json:"latitude"`
	Address   *string `json:"address"`
}

func (r *repository) SaveLocation(ctx context.Context, arg saveLocationRequest) error {
	loc := location{
		Longitude: arg.Longitude,
		Latitude:  arg.Latitude,
		Address:   arg.Address,
	}

	key := fmt.Sprintf("user:%s:location", arg.UserID)
	if err := r.redisClient.JSONSet(ctx, key, "$", loc).Err(); err != nil {
		return err
	}

	return nil
}

var errLocationNotFound = errors.New("location not found")

func (r *repository) GetLocation(ctx context.Context, userID string) (location, error) {
	key := fmt.Sprintf("user:%s:location", userID)
	result, err := r.redisClient.JSONGet(ctx, key).Result()
	if err != nil {
		return location{}, err
	}

	if result == "" {
		return location{}, errLocationNotFound
	}

	var loc location

	if err := json.Unmarshal([]byte(result), &loc); err != nil {
		return location{}, err
	}

	return loc, nil
}
