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

// Request body for the CLI's authentication webhook
type AuthWebhookRequest struct {
	SessionToken string `json:"session_token" validate:"required"`
}
