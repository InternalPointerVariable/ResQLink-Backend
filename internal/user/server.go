package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/InternalPointerVariable/ResQLink-Backend/internal/api"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Server struct {
	repository Repository
}

func NewServer(repository Repository) *Server {
	return &Server{
		repository: repository,
	}
}

type signUpRequest struct {
	Email                 string    `json:"email"`
	Password              string    `json:"password"`
	FirstName             string    `json:"firstName"`
	MiddleName            *string   `json:"middleName"`
	LastName              string    `json:"lastName"`
	BirthDate             time.Time `json:"birthDate"`
	Role                  Role      `json:"role"`
	StatusUpdateFrequency uint      `json:"statusUpdateFrequency"`
	IsLocationShared      bool      `json:"isLocationShared"`
}

func (s *Server) SignUp(w http.ResponseWriter, r *http.Request) api.Response {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var data signUpRequest

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&data); err != nil {
		return api.Response{
			Error:   fmt.Errorf("sign up: %w", err),
			Code:    http.StatusBadRequest,
			Message: "Invalid sign up request.",
		}
	}

	if err := s.repository.SignUp(ctx, data); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return api.Response{
				Error:   fmt.Errorf("sign up: %w", err),
				Code:    http.StatusConflict,
				Message: "User " + data.Email + " already exists.",
			}
		}

		return api.Response{
			Error:   fmt.Errorf("sign up: %w", err),
			Code:    http.StatusInternalServerError,
			Message: "Failed to sign up.",
		}
	}

	return api.Response{
		Code:    http.StatusCreated,
		Message: "Successfully signed up.",
	}
}

type userResponse struct {
	UserID                string    `json:"userId"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
	Email                 string    `json:"email"`
	FirstName             string    `json:"firstName"`
	MiddleName            *string   `json:"middleName"`
	LastName              string    `json:"lastName"`
	BirthDate             time.Time `json:"birthDate"`
	Role                  Role      `json:"role"`
	StatusUpdateFrequency uint      `json:"statusUpdateFrequency"`
	IsLocationShared      bool      `json:"isLocationShared"`
}

type signInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type signInResponse struct {
	User    userResponse `json:"user"`
	Session session      `json:"session"`
}

func (s *Server) SignIn(w http.ResponseWriter, r *http.Request) api.Response {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var data signInRequest

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&data); err != nil {
		return api.Response{
			Error:   fmt.Errorf("sign in: %w", err),
			Code:    http.StatusBadRequest,
			Message: "Invalid sign in request.",
		}
	}

	response, err := s.repository.SignIn(ctx, data)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.Response{
				Error:   fmt.Errorf("sign in: %w", err),
				Code:    http.StatusNotFound,
				Message: "Invalid credentials.",
			}
		}

		if errors.Is(err, errInvalidPassword) {
			return api.Response{
				Error:   fmt.Errorf("sign in: %w", err),
				Code:    http.StatusUnauthorized,
				Message: "Invalid password.",
			}
		}

		return api.Response{
			Error:   fmt.Errorf("sign in: %w", err),
			Code:    http.StatusInternalServerError,
			Message: "Failed to sign in.",
		}
	}

	return api.Response{
		Code:    http.StatusOK,
		Message: "Successfully signed in.",
		Data:    response,
	}
}

func (s *Server) SaveLocation(w http.ResponseWriter, r *http.Request) api.Response {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var data saveLocationRequest

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&data); err != nil {
		return api.Response{
			Error:   fmt.Errorf("save location: %w", err),
			Code:    http.StatusBadRequest,
			Message: "Invalid save location request.",
		}
	}

	if err := s.repository.SaveLocation(ctx, data); err != nil {
		return api.Response{
			Error:   fmt.Errorf("save location: %w", err),
			Code:    http.StatusInternalServerError,
			Message: "Failed to save location.",
		}
	}

	return api.Response{
		Code:    http.StatusCreated,
		Message: "Successfully saved location.",
	}
}

func (s *Server) GetLocation(w http.ResponseWriter, r *http.Request) api.Response {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	userID := r.PathValue("userID")

	loc, err := s.repository.GetLocation(ctx, userID)
	if err != nil {
		if errors.Is(err, errLocationNotFound) {
			return api.Response{
				Error:   fmt.Errorf("get location: %w", err),
				Code:    http.StatusNotFound,
				Message: "Location not found.",
			}
		}

		return api.Response{
			Error:   fmt.Errorf("get location: %w", err),
			Code:    http.StatusInternalServerError,
			Message: "Failed to get location.",
		}
	}

	return api.Response{
		Code:    http.StatusOK,
		Message: "Successfully fetched location.",
		Data:    loc,
	}
}

func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		token, err := r.Cookie("session")
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		_, err = s.repository.validateSessionToken(ctx, token.Value)
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
