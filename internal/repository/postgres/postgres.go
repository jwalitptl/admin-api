package postgres

import (
	"github.com/jmoiron/sqlx"
	"github.com/jwalitptl/admin-api/internal/repository"
)

type accountRepository struct {
	db *sqlx.DB
}

type organizationRepository struct {
	db *sqlx.DB
}

type clinicRepository struct {
	db *sqlx.DB
}

type clinicianRepository struct {
	db *sqlx.DB
}

type appointmentRepository struct {
	db *sqlx.DB
}

type rbacRepository struct {
	db *sqlx.DB
}

func NewAccountRepository(db *sqlx.DB) repository.AccountRepository {
	return &accountRepository{db: db}
}

func NewOrganizationRepository(db *sqlx.DB) repository.OrganizationRepository {
	return &organizationRepository{db: db}
}

func NewClinicRepository(db *sqlx.DB) repository.ClinicRepository {
	return &clinicRepository{db: db}
}

func NewClinicianRepository(db *sqlx.DB) repository.ClinicianRepository {
	return &clinicianRepository{db: db}
}

func NewAppointmentRepository(db *sqlx.DB) repository.AppointmentRepository {
	return &appointmentRepository{db: db}
}

func NewRBACRepository(db *sqlx.DB) repository.RBACRepository {
	return &rbacRepository{db: db}
}
