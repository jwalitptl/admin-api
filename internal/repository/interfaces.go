package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
)

type AccountRepository interface {
	CreateAccount(ctx context.Context, account *model.Account) error
	GetAccount(ctx context.Context, id uuid.UUID) (*model.Account, error)
	UpdateAccount(ctx context.Context, account *model.Account) error
	DeleteAccount(ctx context.Context, id uuid.UUID) error
	ListAccounts(ctx context.Context) ([]*model.Account, error)
}

type OrganizationRepository interface {
	CreateOrganization(ctx context.Context, org *model.Organization) error
	GetOrganization(ctx context.Context, id uuid.UUID) (*model.Organization, error)
	UpdateOrganization(ctx context.Context, org *model.Organization) error
	DeleteOrganization(ctx context.Context, id uuid.UUID) error
	ListOrganizations(ctx context.Context, accountID uuid.UUID) ([]*model.Organization, error)
}

type ClinicRepository interface {
	Create(ctx context.Context, clinic *model.Clinic) error
	Get(ctx context.Context, id uuid.UUID) (*model.Clinic, error)
	Update(ctx context.Context, clinic *model.Clinic) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, organizationID uuid.UUID) ([]*model.Clinic, error)
}

type ClinicianRepository interface {
	CreateClinician(ctx context.Context, clinician *model.Clinician) error
	GetClinician(ctx context.Context, id uuid.UUID) (*model.Clinician, error)
	UpdateClinician(ctx context.Context, clinician *model.Clinician) error
	DeleteClinician(ctx context.Context, id uuid.UUID) error
	ListClinicians(ctx context.Context) ([]*model.Clinician, error)
	ListClinicClinicians(ctx context.Context, clinicID uuid.UUID) ([]*model.Clinician, error)
	ListClinicianClinics(ctx context.Context, clinicianID uuid.UUID) ([]*model.Clinic, error)
	GetByEmail(ctx context.Context, email string) (*model.Clinician, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, hashedPassword string) error
	VerifyPassword(ctx context.Context, email, password string) (*model.Clinician, error)
	AssignToClinic(ctx context.Context, clinicianID, clinicID uuid.UUID) error
	RemoveFromClinic(ctx context.Context, clinicianID, clinicID uuid.UUID) error
	AssignRole(ctx context.Context, clinicianID, roleID uuid.UUID) error
	RemoveRole(ctx context.Context, clinicianID, roleID uuid.UUID) error
	ListClinicianRoles(ctx context.Context, clinicianID uuid.UUID) ([]*model.Role, error)
	GetRole(ctx context.Context, roleID uuid.UUID) (*model.Role, error)
	AssignRoleToClinician(ctx context.Context, clinicianID, roleID, organizationID uuid.UUID) error
}

type AppointmentRepository interface {
	Create(ctx context.Context, appointment *model.Appointment) error
	Get(ctx context.Context, id uuid.UUID) (*model.Appointment, error)
	Update(ctx context.Context, appointment *model.Appointment) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, clinicID uuid.UUID, filters map[string]interface{}) ([]*model.Appointment, error)
	CheckConflicts(ctx context.Context, clinicianID uuid.UUID, startTime, endTime time.Time, excludeID *uuid.UUID) (bool, error)
	GetClinicianAppointments(ctx context.Context, clinicianID uuid.UUID, startDate, endDate time.Time) ([]*model.Appointment, error)
}

type RBACRepository interface {
	CreateRole(ctx context.Context, role *model.Role) error
	GetRole(ctx context.Context, id uuid.UUID) (*model.Role, error)
	UpdateRole(ctx context.Context, role *model.Role) error
	DeleteRole(ctx context.Context, id uuid.UUID) error
	ListRoles(ctx context.Context, organizationID *uuid.UUID) ([]*model.Role, error)

	CreatePermission(ctx context.Context, permission *model.Permission) error
	GetPermission(ctx context.Context, id uuid.UUID) (*model.Permission, error)
	UpdatePermission(ctx context.Context, permission *model.Permission) error
	DeletePermission(ctx context.Context, id uuid.UUID) error
	ListPermissions(ctx context.Context) ([]*model.Permission, error)

	AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error
	RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error
	ListRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*model.Permission, error)

	AssignRoleToClinician(ctx context.Context, clinicianID, roleID, organizationID uuid.UUID) error
	RemoveRoleFromClinician(ctx context.Context, clinicianID, roleID, organizationID uuid.UUID) error
	ListClinicianRoles(ctx context.Context, clinicianID, orgID uuid.UUID) ([]*model.Role, error)
	HasPermission(ctx context.Context, clinicianID uuid.UUID, permission string, organizationID uuid.UUID) (bool, error)

	AssignRole(ctx context.Context, clinicianID, roleID uuid.UUID) error
	RemoveRole(ctx context.Context, clinicianID, roleID uuid.UUID) error
}

type PatientRepository interface {
	Create(ctx context.Context, patient *model.Patient) error
	Get(ctx context.Context, id uuid.UUID) (*model.Patient, error)
	Update(ctx context.Context, patient *model.Patient) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, clinicID uuid.UUID) ([]*model.Patient, error)
	DeletePatientAppointments(ctx context.Context, patientID uuid.UUID) error
}

// ... other repository interfaces
