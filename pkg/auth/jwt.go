package auth

import (
	"github.com/jwalitptl/admin-api/internal/model"
)

type JWTService interface {
	GenerateAccessToken(user *model.User) (string, error)
	GenerateRefreshToken(user *model.User) (string, error)
	ValidateToken(token string) (map[string]interface{}, error)
	ValidateRefreshToken(token string) (*model.TokenClaims, error)
}

type jwtService struct {
	secret string
}

func NewJWTService(secret string) JWTService {
	return &jwtService{
		secret: secret,
	}
}

func (s *jwtService) GenerateAccessToken(user *model.User) (string, error) {
	// Implementation here
	return "", nil
}

func (s *jwtService) GenerateRefreshToken(user *model.User) (string, error) {
	// Implementation here
	return "", nil
}

func (s *jwtService) ValidateToken(token string) (map[string]interface{}, error) {
	// Implementation here
	return nil, nil
}

func (s *jwtService) ValidateRefreshToken(token string) (*model.TokenClaims, error) {
	// Implementation here
	return nil, nil
}
