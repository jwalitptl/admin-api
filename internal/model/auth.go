package model

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type TokenClaims struct {
	ID             string   `json:"id"`
	ClinicianID    string   `json:"clinician_id"`
	Email          string   `json:"email"`
	Roles          []string `json:"roles"`
	Permissions    []string `json:"permissions"`
	Organization   string   `json:"organization,omitempty"`
	TokenType      string   `json:"token_type"`
	RefreshTokenID string   `json:"refresh_token_id,omitempty"`
}
