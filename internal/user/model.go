package user

import "time"

type role = string

const (
	citizen   role = "citizen"
	responder role = "responder"
)

type session struct {
	SessionID string    `json:"sessionId"`
	UserID    string    `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
}
