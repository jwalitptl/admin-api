package postgres

import (
	"github.com/jmoiron/sqlx"
	"github.com/jwalitptl/admin-api/internal/repository"
)

// NewRepositories creates all repositories
func NewRepositories(db *sqlx.DB) *Repositories {
	base := NewBaseRepository(db)

	return &Repositories{
		Account:       NewAccountRepository(base),
		Organization:  NewOrganizationRepository(base),
		User:          NewUserRepository(base),
		Appointment:   NewAppointmentRepository(base),
		Patient:       NewPatientRepository(base),
		RBAC:          NewRBACRepository(base),
		Audit:         NewAuditRepository(base),
		Token:         NewTokenRepository(base),
		Region:        NewRegionRepository(base),
		MedicalRecord: NewMedicalRecordRepository(base),
	}
}

// Repositories holds all repository implementations
type Repositories struct {
	Account       repository.AccountRepository
	Organization  repository.OrganizationRepository
	User          repository.UserRepository
	Appointment   repository.AppointmentRepository
	Patient       repository.PatientRepository
	RBAC          repository.RBACRepository
	Audit         repository.AuditRepository
	Token         repository.TokenRepository
	Region        repository.RegionRepository
	MedicalRecord repository.MedicalRecordRepository
}
