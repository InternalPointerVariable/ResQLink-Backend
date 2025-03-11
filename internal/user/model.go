package user

import "time"

type Role = string

const (
	Citizen   Role = "citizen"
	Responder Role = "responder"
)

type session struct {
	SessionID string    `json:"sessionId"`
	UserID    string    `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
}
