package models

// Request body for `/authenticate`
type AuthenticateRequest struct {
	Token     string `json:"token" validate:"required"`
	TokenType string `json:"token_type" validate:"required"`
}

// Response body for `/authenticate`
type AuthenticateResponse struct {
	SessionToken string `json:"session_token"`
}
