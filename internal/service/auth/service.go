package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/email"
	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
	"github.com/jwalitptl/admin-api/internal/service/audit"
	"github.com/jwalitptl/admin-api/pkg/auth"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenGeneration    = errors.New("failed to generate token")
)

const (
	tokenExpiry       = 24 * time.Hour
	resetTokenExpiry  = 1 * time.Hour
	verifyTokenExpiry = 48 * time.Hour
	maxLoginAttempts  = 5
	lockoutDuration   = 15 * time.Minute
	bcryptCost        = 12
)

type Service struct {
	userRepo  repository.UserRepository
	jwtSvc    auth.JWTService
	tokenRepo repository.TokenRepository
	emailSvc  email.Service
	auditor   *audit.Service
}

func NewService(userRepo repository.UserRepository, jwtSvc auth.JWTService,
	tokenRepo repository.TokenRepository, emailSvc email.Service, auditor *audit.Service) *Service {
	return &Service{
		userRepo:  userRepo,
		jwtSvc:    jwtSvc,
		tokenRepo: tokenRepo,
		emailSvc:  emailSvc,
		auditor:   auditor,
	}
}

func (s *Service) Login(ctx context.Context, email, password string) (*model.TokenResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if user.Status == model.UserStatusLocked {
		if time.Since(user.LastLoginAttempt) < lockoutDuration {
			return nil, fmt.Errorf("account is locked, please try again later")
		}
		user.Status = model.UserStatusActive
		user.LoginAttempts = 0
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		user.LoginAttempts++
		user.LastLoginAttempt = time.Now()

		if user.LoginAttempts >= maxLoginAttempts {
			user.Status = model.UserStatusLocked
		}

		if err := s.userRepo.Update(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to update login attempts: %w", err)
		}

		return nil, fmt.Errorf("invalid credentials")
	}

	// Reset login attempts on successful login
	user.LoginAttempts = 0
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update login timestamp: %w", err)
	}

	tokens, err := s.generateTokens(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	s.auditor.Log(ctx, user.ID, user.OrganizationID, "login", "auth", user.ID, &audit.LogOptions{
		Metadata: map[string]interface{}{
			"email": user.Email,
		},
	})

	return tokens, nil
}

func (s *Service) ValidateToken(ctx context.Context, token string) (*model.TokenClaims, error) {
	claims, err := s.jwtSvc.ValidateToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	email, ok := claims["email"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token")
	}

	return &model.TokenClaims{
		UserID: parsedUserID,
		Email:  email,
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*model.TokenResponse, error) {
	claims, err := s.jwtSvc.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	user, err := s.userRepo.Get(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	tokens, err := s.generateTokens(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	s.auditor.Log(ctx, user.ID, user.OrganizationID, "refresh_token", "auth", user.ID, nil)

	return tokens, nil
}

func (s *Service) RevokeToken(ctx context.Context, token string) error {
	// Implementation of RevokeToken method
	return nil
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

func (s *Service) Register(ctx context.Context, req *model.RegisterRequest) (*model.User, error) {
	// Check if user already exists
	existing, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, fmt.Errorf("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		Base: model.Base{
			ID: uuid.New(),
		},
		PasswordHash: string(hashedPassword),
		Name:         fmt.Sprintf("%s %s", req.FirstName, req.LastName),
		Email:        req.Email,
		Phone:        req.Phone,
		Status:       "pending",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Send verification email
	if err := s.sendVerificationEmail(ctx, user); err != nil {
		// Log error but don't fail registration
		log.Printf("failed to send verification email: %v", err)
	}

	// Generate verification token
	token := uuid.New().String()
	if err := s.tokenRepo.StoreVerificationToken(ctx, user.ID, token, time.Now().Add(verifyTokenExpiry)); err != nil {
		return nil, fmt.Errorf("failed to store verification token: %w", err)
	}

	s.auditor.Log(ctx, user.ID, user.OrganizationID, "register", "auth", user.ID, &audit.LogOptions{
		Metadata: map[string]interface{}{
			"email": user.Email,
		},
	})

	return user, nil
}

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil // User not found, but return nil for security
	}

	// Generate reset token
	token := uuid.New().String()
	expiry := time.Now().Add(24 * time.Hour)

	// Store reset token
	if err := s.tokenRepo.StoreResetToken(ctx, user.ID, token, expiry); err != nil {
		return fmt.Errorf("failed to store reset token: %w", err)
	}

	// Send reset email
	if err := s.emailSvc.SendPasswordReset(user.Email, token); err != nil {
		return fmt.Errorf("failed to send reset email: %w", err)
	}

	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	userID, err := s.tokenRepo.ValidateResetToken(ctx, token)
	if err != nil {
		return fmt.Errorf("invalid or expired reset token")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.userRepo.Get(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	user.PasswordHash = string(hashedPassword)
	user.UpdatedAt = time.Now()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	s.auditor.Log(ctx, user.ID, user.OrganizationID, "reset_password", "auth", user.ID, nil)

	return nil
}

func (s *Service) ResendVerification(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if user.EmailVerified {
		return fmt.Errorf("email already verified")
	}

	// Generate new verification token
	token := uuid.New().String()
	expiry := time.Now().Add(24 * time.Hour)

	// Store verification token
	if err := s.tokenRepo.StoreVerificationToken(ctx, user.ID, token, expiry); err != nil {
		return fmt.Errorf("failed to store verification token: %w", err)
	}

	// Send verification email
	if err := s.emailSvc.SendVerification(user.Email, token); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	return nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.tokenRepo.InvalidateToken(ctx, token)
}

func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	return s.verifyEmailToken(ctx, token)
}

func (s *Service) verifyEmailToken(ctx context.Context, token string) error {
	userID, err := s.tokenRepo.ValidateVerificationToken(ctx, token)
	if err != nil {
		return fmt.Errorf("invalid or expired verification token")
	}

	if err := s.userRepo.UpdateEmailVerified(ctx, userID, true); err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	return s.tokenRepo.InvalidateVerificationToken(ctx, token)
}

func (s *Service) generateTokens(user *model.User) (*model.TokenResponse, error) {
	accessToken, err := s.jwtSvc.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtSvc.GenerateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	return &model.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *Service) sendVerificationEmail(ctx context.Context, user *model.User) error {
	token := uuid.New().String()
	if err := s.tokenRepo.StoreVerificationToken(ctx, user.ID, token, time.Now().Add(verifyTokenExpiry)); err != nil {
		return fmt.Errorf("failed to store verification token: %w", err)
	}

	return s.emailSvc.SendVerification(user.Email, token)
}
