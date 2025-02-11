package auth

import (
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
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
	claims := jwt.MapClaims{
		"user_id":         user.ID.String(),
		"email":           user.Email,
		"type":            user.Type,
		"organization_id": user.OrganizationID.String(),
		"exp":             time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secret))
}

func (s *jwtService) GenerateRefreshToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secret))
}

func (s *jwtService) ValidateToken(token string) (map[string]interface{}, error) {
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok && parsedToken.Valid {
		log.Printf("JWT Validation: claims=%#v", claims)
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

func (s *jwtService) ValidateRefreshToken(token string) (*model.TokenClaims, error) {
	claims, err := s.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid user ID in token")
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format")
	}

	email, ok := claims["email"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid email in token")
	}

	return &model.TokenClaims{
		UserID: parsedUserID.String(),
		Email:  email,
	}, nil
}
