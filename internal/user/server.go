package user

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/InternalPointerVariable/ResQLink-Backend/internal/api"
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
		return api.Response{
			Error:   fmt.Errorf("sign in: %w", err),
			Code:    http.StatusInternalServerError,
			Message: "Failed to sign in.",
		}
	}

	// TODO: Return session token for the client

	return api.Response{
		Code:    http.StatusOK,
		Message: "Successfully signed in.",
		Data:    response,
	}
}
