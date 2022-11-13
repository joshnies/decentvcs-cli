package models

import (
	"time"
)

type AccessKey struct {
	ID        string    `json:"_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	// ID of the user in our auth provider.
	UserID string `json:"user_id"`
	// ID of the team that owns the project.
	TeamID string `json:"team_id,omitempty" `
	Scope  string `json:"scope" `
}
