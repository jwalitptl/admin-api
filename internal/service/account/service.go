package account

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/email"
	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
	"github.com/jwalitptl/admin-api/internal/service/audit"
)

type AccountServicer interface {
	CreateAccount(ctx context.Context, req *model.CreateAccountRequest) (*model.Account, error)
	GetAccount(ctx context.Context, id uuid.UUID) (*model.Account, error)
	UpdateAccount(ctx context.Context, account *model.Account) error
	DeleteAccount(ctx context.Context, id uuid.UUID) error
	ListAccounts(ctx context.Context, filters *model.AccountFilters) ([]*model.Account, error)
	CreateOrganization(ctx context.Context, org *model.Organization) error
	UpdateOrganization(ctx context.Context, org *model.Organization) error
	DeleteOrganization(ctx context.Context, id uuid.UUID) error
	ListOrganizations(ctx context.Context, accountID uuid.UUID) ([]*model.Organization, error)
	GetOrganization(ctx context.Context, id uuid.UUID) (*model.Organization, error)
	// ... other methods used by the handler
}

type Service struct {
	accountRepo repository.AccountRepository
	orgRepo     repository.OrganizationRepository
	emailSvc    email.Service
	auditor     *audit.Service
}

func NewService(accountRepo repository.AccountRepository, orgRepo repository.OrganizationRepository, emailSvc email.Service, auditor *audit.Service) *Service {
	return &Service{
		accountRepo: accountRepo,
		orgRepo:     orgRepo,
		emailSvc:    emailSvc,
		auditor:     auditor,
	}
}

func (s *Service) CreateAccount(ctx context.Context, req *model.CreateAccountRequest) (*model.Account, error) {
	if err := s.validateAccountRequest(req); err != nil {
		return nil, fmt.Errorf("invalid account request: %w", err)
	}

	account := &model.Account{
		ID:        uuid.New(),
		Name:      req.Name,
		Email:     req.Email,
		Status:    string(model.AccountStatusActive),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.accountRepo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	// Create default organization
	org := &model.Organization{
		AccountID: account.ID.String(),
		Name:      req.Name,
		Status:    string(model.OrganizationStatusActive),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.orgRepo.CreateOrganization(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, org.ID, "create", "account", account.ID, &audit.LogOptions{
		Changes: account,
	})

	// Send welcome email
	if err := s.emailSvc.SendWelcome(ctx, account.Email, account.Name); err != nil {
		s.auditor.Log(ctx, uuid.Nil, org.ID, "welcome_email_failed", "account", account.ID, &audit.LogOptions{
			Metadata: map[string]interface{}{
				"error": err.Error(),
			},
		})
	}

	return account, nil
}

func (s *Service) GetAccount(ctx context.Context, id uuid.UUID) (*model.Account, error) {
	account, err := s.accountRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	org, err := s.orgRepo.GetOrganization(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, org.ID, "read", "account", id, nil)
	return account, nil
}

func (s *Service) UpdateAccount(ctx context.Context, account *model.Account) error {
	if err := s.validateAccount(account); err != nil {
		return fmt.Errorf("invalid account: %w", err)
	}

	account.UpdatedAt = time.Now()
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	org, err := s.orgRepo.GetOrganization(ctx, account.ID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, org.ID, "update", "account", account.ID, &audit.LogOptions{
		Changes: account,
	})

	return nil
}

func (s *Service) DeleteAccount(ctx context.Context, id uuid.UUID) error {
	account, err := s.accountRepo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	org, err := s.orgRepo.GetOrganization(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	if err := s.accountRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, org.ID, "delete", "account", id, &audit.LogOptions{
		Changes: account,
	})

	return nil
}

func (s *Service) ListAccounts(ctx context.Context, filters *model.AccountFilters) ([]*model.Account, error) {
	return s.accountRepo.List(ctx)
}

func (s *Service) validateAccountRequest(req *model.CreateAccountRequest) error {
	if req.Name == "" {
		return fmt.Errorf("account name is required")
	}

	if req.Email == "" {
		return fmt.Errorf("email is required")
	}

	return nil
}

func (s *Service) validateAccount(account *model.Account) error {
	if account.Name == "" {
		return fmt.Errorf("account name is required")
	}

	if account.Email == "" {
		return fmt.Errorf("email is required")
	}

	if account.Status == "" {
		return fmt.Errorf("status is required")
	}

	return nil
}

// Organization methods
func (s *Service) CreateOrganization(ctx context.Context, org *model.Organization) error {
	org.ID = uuid.New()
	org.CreatedAt = time.Now()
	org.UpdatedAt = time.Now()

	if err := s.orgRepo.CreateOrganization(ctx, org); err != nil {
		return fmt.Errorf("failed to create organization: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, org.ID, "create", "organization", org.ID, &audit.LogOptions{
		Changes: org,
	})
	return nil
}

func (s *Service) GetOrganization(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
	org, err := s.orgRepo.GetOrganization(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, org.ID, "read", "organization", id, nil)
	return org, nil
}

func (s *Service) UpdateOrganization(ctx context.Context, org *model.Organization) error {
	org.UpdatedAt = time.Now()
	if err := s.orgRepo.UpdateOrganization(ctx, org); err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, org.ID, "update", "organization", org.ID, &audit.LogOptions{
		Changes: org,
	})
	return nil
}

func (s *Service) DeleteOrganization(ctx context.Context, id uuid.UUID) error {
	if err := s.orgRepo.DeleteOrganization(ctx, id); err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, id, "delete", "organization", id, nil)
	return nil
}

func (s *Service) ListOrganizations(ctx context.Context, accountID uuid.UUID) ([]*model.Organization, error) {
	return s.orgRepo.ListOrganizations(ctx, accountID)
}
