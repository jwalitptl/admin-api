package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenGeneration    = errors.New("failed to generate token")
)

type Service interface {
	Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error)
	RefreshToken(ctx context.Context, req *model.RefreshTokenRequest) (*model.LoginResponse, error)
	ValidateToken(ctx context.Context, token string) (*model.TokenClaims, error)
	RevokeToken(ctx context.Context, token string) error
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Claims struct {
	ClinicianID string `json:"clinician_id"`
	Email       string `json:"email"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	ClinicianID string `json:"clinician_id"`
	TokenID     string `json:"token_id"`
	jwt.RegisteredClaims
}

type JWTConfig struct {
	Secret             string
	RefreshSecret      string
	ExpiryHours        int
	RefreshExpiryHours int
}

type ClinicianRepository interface {
	GetByEmail(ctx context.Context, email string) (*model.Clinician, error)
	GetClinician(ctx context.Context, id uuid.UUID) (*model.Clinician, error)
	VerifyPassword(ctx context.Context, email, password string) (*model.Clinician, error)
}

type service struct {
	clinicianRepo   ClinicianRepository
	jwtSecret       []byte
	refreshSecret   []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewService(clinicianRepo ClinicianRepository, config JWTConfig) Service {
	return &service{
		clinicianRepo:   clinicianRepo,
		jwtSecret:       []byte(config.Secret),
		refreshSecret:   []byte(config.RefreshSecret),
		accessTokenTTL:  time.Duration(config.ExpiryHours) * time.Hour,
		refreshTokenTTL: time.Duration(config.RefreshExpiryHours) * time.Hour,
	}
}

func (s *service) Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error) {
	// Add debug logging
	fmt.Printf("Login attempt for email: %s\n", req.Email)

	clinician, err := s.clinicianRepo.VerifyPassword(ctx, req.Email, req.Password)
	if err != nil {
		fmt.Printf("Password verification failed: %v\n", err) // Debug log
		return nil, fmt.Errorf("invalid email or password")
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(clinician)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(clinician)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &model.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *service) generateAccessToken(clinician *model.Clinician) (string, error) {
	// Generate access token
	accessClaims := &Claims{
		ClinicianID: clinician.ID.String(),
		Email:       clinician.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	signedAccessToken, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", ErrTokenGeneration
	}

	return signedAccessToken, nil
}

func (s *service) generateRefreshToken(clinician *model.Clinician) (string, error) {
	// Generate refresh token
	tokenID := uuid.New().String()
	refreshClaims := &RefreshClaims{
		ClinicianID: clinician.ID.String(),
		TokenID:     tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	signedRefreshToken, err := refreshToken.SignedString(s.refreshSecret)
	if err != nil {
		return "", ErrTokenGeneration
	}

	return signedRefreshToken, nil
}

func (s *service) ValidateToken(ctx context.Context, tokenStr string) (*model.TokenClaims, error) {
	parsedToken, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := parsedToken.Claims.(*Claims); ok && parsedToken.Valid {
		return &model.TokenClaims{
			ID:          claims.ID,
			ClinicianID: claims.ClinicianID,
			Email:       claims.Email,
		}, nil
	}

	return nil, errors.New("invalid token")
}

func (s *service) RefreshToken(ctx context.Context, req *model.RefreshTokenRequest) (*model.LoginResponse, error) {
	token, err := jwt.ParseWithClaims(req.RefreshToken, &RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.refreshSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*RefreshClaims); ok && token.Valid {
		clinicianID, err := uuid.Parse(claims.ClinicianID)
		if err != nil {
			return nil, err
		}

		clinician, err := s.clinicianRepo.GetClinician(context.Background(), clinicianID)
		if err != nil {
			return nil, err
		}

		return s.generateTokenPair(clinician)
	}

	return nil, errors.New("invalid refresh token")
}

func (s *service) RevokeToken(ctx context.Context, token string) error {
	// Implementation of RevokeToken method
	return nil
}

func (s *service) generateTokenPair(clinician *model.Clinician) (*model.LoginResponse, error) {
	accessToken, err := s.generateAccessToken(clinician)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateRefreshToken(clinician)
	if err != nil {
		return nil, err
	}

	return &model.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
