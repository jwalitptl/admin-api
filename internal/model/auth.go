package model

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// AuthRequest types
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type RegisterRequest struct {
	Email          string    `json:"email" binding:"required,email"`
	Password       string    `json:"password" binding:"required,min=8"`
	FirstName      string    `json:"first_name" binding:"required"`
	LastName       string    `json:"last_name" binding:"required"`
	Phone          string    `json:"phone" binding:"required"`
	Type           string    `json:"type" binding:"required,oneof=admin staff provider support patient"`
	Status         string    `json:"status" binding:"required"`
	OrganizationID uuid.UUID `json:"organization_id" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// AuthResponse types
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Auth errors
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// TokenClaims represents JWT claims
type TokenClaims struct {
	jwt.StandardClaims
	UserID         string   `json:"user_id"`
	OrganizationID string   `json:"organization_id"`
	Email          string   `json:"email"`
	Type           string   `json:"type"`
	Roles          []string `json:"roles"`
	Permissions    []string `json:"permissions"`
}

type UserToken struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	Type      string    `json:"type" db:"type"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
