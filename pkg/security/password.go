package security

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrHashingFailed = errors.New("password hashing failed")
	MinPasswordLen   = 8
)

// PasswordHasher provides interface for password operations
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hashedPassword, password string) error
}

type bcryptHasher struct {
	cost int
}

// NewBcryptHasher creates a new password hasher using bcrypt
func NewBcryptHasher(cost int) PasswordHasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	return &bcryptHasher{cost: cost}
}

func (b *bcryptHasher) Hash(password string) (string, error) {
	if len(password) < MinPasswordLen {
		return "", errors.New("password too short")
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	if err != nil {
		return "", ErrHashingFailed
	}
	return string(bytes), nil
}

func (b *bcryptHasher) Compare(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
