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
