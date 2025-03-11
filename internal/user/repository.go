package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Repository interface {
	SignUp(ctx context.Context, arg signUpRequest) error
	SignIn(ctx context.Context, arg signInRequest) (signInResponse, error)

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

func (r *repository) generateSessionToken() (string, error) {
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	token := encoder.EncodeToString(bytes)

	return token, nil
}

func (r *repository) createSession(ctx context.Context, token, userID string) (session, error) {
	hash := sha256.Sum256([]byte(token))
	sessionID := hex.EncodeToString(hash[:])

	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	ses := session{
		SessionID: sessionID,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}

	byt, err := json.Marshal(ses)
	if err != nil {
		return session{}, err
	}

	sessionKey := fmt.Sprintf("session:%s", sessionID)
	if err := r.redisClient.Set(ctx, sessionKey, string(byt), time.Until(expiresAt)).Err(); err != nil {
		return session{}, err
	}

	userSessionsKey := fmt.Sprintf("user_sessions:%s", userID)
	if err := r.redisClient.SAdd(ctx, userSessionsKey, sessionID).Err(); err != nil {
		return session{}, err
	}

	return ses, nil
}

func (r *repository) validateSessionToken(ctx context.Context, token string) (session, error) {
	hash := sha256.Sum256([]byte(token))
	sessionID := hex.EncodeToString(hash[:])

	sessionKey := fmt.Sprintf("session:%s", sessionID)

	data, err := r.redisClient.Get(ctx, sessionKey).Result()
	if err != nil {
		return session{}, err
	}

	var ses session

	if err := json.Unmarshal([]byte(data), &ses); err != nil {
		return session{}, err
	}

	now := time.Now()
	if now.After(ses.ExpiresAt) || now.Equal(ses.ExpiresAt) {
		if err := r.redisClient.Del(ctx, sessionKey).Err(); err != nil {
			return session{}, err
		}

		userSessionsKey := fmt.Sprintf("user_sessions:%s", ses.UserID)
		if err := r.redisClient.SRem(ctx, userSessionsKey, sessionID).Err(); err != nil {
			return session{}, err
		}
	}

	// If session is close to expiration (3 days), extend it
	beforeExpiry := ses.ExpiresAt.Add(-3 * 24 * time.Hour)
	if now.After(beforeExpiry) || now.Equal(beforeExpiry) {
		ses.ExpiresAt = now.Add(7 * 24 * time.Hour)

		if err := r.redisClient.Set(
			ctx,
			sessionKey,
			ses,
			time.Until(ses.ExpiresAt),
		).Err(); err != nil {
			return session{}, err
		}
	}

	return ses, nil
}

func (r *repository) invalidateSession(ctx context.Context, sessionID, userID string) error {
	sessionKey := fmt.Sprintf("session:%s", sessionID)
	if err := r.redisClient.Del(ctx, sessionKey).Err(); err != nil {
		return err
	}

	userSessionsKey := fmt.Sprintf("user_sessions:%s", userID)
	if err := r.redisClient.SRem(ctx, userSessionsKey, sessionID).Err(); err != nil {
		return err
	}

	return nil
}
