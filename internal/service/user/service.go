package user

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/jwalitptl/admin-api/internal/email"
	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
	"github.com/jwalitptl/admin-api/internal/service/audit"
)

const (
	bcryptCost        = 12
	maxLoginAttempts  = 5
	lockoutDuration   = 15 * time.Minute
	tokenExpiry       = 24 * time.Hour
	verifyTokenExpiry = 48 * time.Hour
)

type UserServicer interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUser(ctx context.Context, id uuid.UUID) (*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ListUsers(ctx context.Context, filters *model.UserFilters) ([]*model.User, error)
	AssignToClinic(ctx context.Context, userID, clinicID uuid.UUID) error
	RemoveFromClinic(ctx context.Context, userID, clinicID uuid.UUID) error
	ListUserClinics(ctx context.Context, userID uuid.UUID) ([]*model.Clinic, error)
	AssignRole(ctx context.Context, userID, roleID uuid.UUID) error
	RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error
	ListUserRoles(ctx context.Context, userID uuid.UUID) ([]*model.Role, error)
}

type Service struct {
	repo      repository.UserRepository
	emailSvc  email.Service
	auditor   *audit.Service
	tokenRepo repository.TokenRepository
}

func NewService(repo repository.UserRepository, emailSvc email.Service, tokenRepo repository.TokenRepository, auditor *audit.Service) *Service {
	return &Service{
		repo:      repo,
		emailSvc:  emailSvc,
		tokenRepo: tokenRepo,
		auditor:   auditor,
	}
}

func (s *Service) CreateUser(ctx context.Context, user *model.User) error {
	if err := s.validateUser(user); err != nil {
		return fmt.Errorf("invalid user data: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.ID = uuid.New()
	user.PasswordHash = string(hashedPassword)
	user.Status = model.UserStatusActive
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	if err := s.repo.Create(ctx, user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Send verification email
	token := uuid.New().String()
	if err := s.tokenRepo.StoreVerificationToken(ctx, user.ID, token, time.Now().Add(verifyTokenExpiry)); err != nil {
		return fmt.Errorf("failed to store verification token: %w", err)
	}

	if err := s.emailSvc.SendVerification(ctx, user.Email, token); err != nil {
		s.auditor.Log(ctx, user.ID, user.OrganizationID, "verification_email_failed", "user", user.ID, &audit.LogOptions{
			Metadata: map[string]interface{}{
				"error": err.Error(),
			},
		})
	}

	s.auditor.Log(ctx, user.ID, user.OrganizationID, "create", "user", user.ID, &audit.LogOptions{
		Changes: user,
	})

	return nil
}

func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	s.auditor.Log(ctx, id, user.OrganizationID, "read", "user", id, nil)
	return user, nil
}

func (s *Service) UpdateUser(ctx context.Context, user *model.User) error {
	if err := s.validateUser(user); err != nil {
		return fmt.Errorf("invalid user data: %w", err)
	}

	user.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	s.auditor.Log(ctx, user.ID, user.OrganizationID, "update", "user", user.ID, &audit.LogOptions{
		Changes: user,
	})

	return nil
}

func (s *Service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	user, err := s.repo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	s.auditor.Log(ctx, id, user.OrganizationID, "delete", "user", id, nil)
	return nil
}

func (s *Service) ListUsers(ctx context.Context, filters *model.UserFilters) ([]*model.User, error) {
	users, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	userID, err := s.tokenRepo.ValidateVerificationToken(ctx, token)
	if err != nil {
		return fmt.Errorf("invalid or expired verification token: %w", err)
	}

	if err := s.repo.UpdateEmailVerified(ctx, userID, true); err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	if err := s.tokenRepo.InvalidateVerificationToken(ctx, token); err != nil {
		return fmt.Errorf("failed to invalidate token: %w", err)
	}

	user, err := s.repo.Get(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	s.auditor.Log(ctx, userID, user.OrganizationID, "verify_email", "user", userID, nil)
	return nil
}

func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil // Don't reveal if email exists
	}

	token := uuid.New().String()
	if err := s.tokenRepo.StoreResetToken(ctx, user.ID, token, time.Now().Add(tokenExpiry)); err != nil {
		return fmt.Errorf("failed to store reset token: %w", err)
	}

	if err := s.emailSvc.SendPasswordReset(ctx, email, token); err != nil {
		s.auditor.Log(ctx, user.ID, user.OrganizationID, "reset_email_failed", "user", user.ID, &audit.LogOptions{
			Metadata: map[string]interface{}{
				"error": err.Error(),
			},
		})
		return fmt.Errorf("failed to send reset email: %w", err)
	}

	s.auditor.Log(ctx, user.ID, user.OrganizationID, "request_reset", "user", user.ID, nil)
	return nil
}

func (s *Service) validateUser(user *model.User) error {
	if user.Email == "" {
		return fmt.Errorf("email is required")
	}

	if user.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization ID is required")
	}

	if user.FirstName == "" {
		return fmt.Errorf("first name is required")
	}

	if user.LastName == "" {
		return fmt.Errorf("last name is required")
	}

	return nil
}

func (s *Service) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	// Implementation of AssignRole method
	return nil
}

func (s *Service) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	// Implementation of RemoveRole method
	return nil
}

func (s *Service) ListUserRoles(ctx context.Context, userID uuid.UUID) ([]*model.Role, error) {
	// Implementation of ListUserRoles method
	return nil, nil
}

func (s *Service) AssignToClinic(ctx context.Context, userID, clinicID uuid.UUID) error {
	return s.repo.AssignToClinic(ctx, userID, clinicID)
}

func (s *Service) RemoveFromClinic(ctx context.Context, userID, clinicID uuid.UUID) error {
	// Implementation of RemoveFromClinic method
	return nil
}

func (s *Service) ListUserClinics(ctx context.Context, userID uuid.UUID) ([]*model.Clinic, error) {
	// Implementation of ListUserClinics method
	return nil, nil
}
