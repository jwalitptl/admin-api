package account

import (
	"context"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
)

type Service interface {
	// Account methods
	CreateAccount(ctx context.Context, account *model.Account) error
	GetAccount(ctx context.Context, id uuid.UUID) (*model.Account, error)
	UpdateAccount(ctx context.Context, account *model.Account) error
	DeleteAccount(ctx context.Context, id uuid.UUID) error
	ListAccounts(ctx context.Context) ([]*model.Account, error)

	// Organization methods
	CreateOrganization(ctx context.Context, org *model.Organization) error
	GetOrganization(ctx context.Context, id uuid.UUID) (*model.Organization, error)
	UpdateOrganization(ctx context.Context, org *model.Organization) error
	DeleteOrganization(ctx context.Context, id uuid.UUID) error
	ListOrganizations(ctx context.Context, accountID uuid.UUID) ([]*model.Organization, error)
}

type service struct {
	accountRepo      repository.AccountRepository
	organizationRepo repository.OrganizationRepository
}

func NewService(accountRepo repository.AccountRepository, organizationRepo repository.OrganizationRepository) Service {
	return &service{
		accountRepo:      accountRepo,
		organizationRepo: organizationRepo,
	}
}

func (s *service) CreateAccount(ctx context.Context, acc *model.Account) error {
	return s.accountRepo.CreateAccount(ctx, acc)
}

func (s *service) GetAccount(ctx context.Context, id uuid.UUID) (*model.Account, error) {
	return s.accountRepo.GetAccount(ctx, id)
}

func (s *service) UpdateAccount(ctx context.Context, acc *model.Account) error {
	return s.accountRepo.UpdateAccount(ctx, acc)
}

func (s *service) DeleteAccount(ctx context.Context, id uuid.UUID) error {
	return s.accountRepo.DeleteAccount(ctx, id)
}

func (s *service) ListAccounts(ctx context.Context) ([]*model.Account, error) {
	return s.accountRepo.ListAccounts(ctx)
}

func (s *service) CreateOrganization(ctx context.Context, org *model.Organization) error {
	return s.organizationRepo.CreateOrganization(ctx, org)
}

func (s *service) GetOrganization(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
	return s.organizationRepo.GetOrganization(ctx, id)
}

func (s *service) UpdateOrganization(ctx context.Context, org *model.Organization) error {
	return s.organizationRepo.UpdateOrganization(ctx, org)
}

func (s *service) DeleteOrganization(ctx context.Context, id uuid.UUID) error {
	return s.organizationRepo.DeleteOrganization(ctx, id)
}

func (s *service) ListOrganizations(ctx context.Context, accountID uuid.UUID) ([]*model.Organization, error) {
	return s.organizationRepo.ListOrganizations(ctx, accountID)
}
