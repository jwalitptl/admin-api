package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
)

// All repository interfaces in one file
type (
	// AccountRepository handles account operations
	AccountRepository interface {
		Create(ctx context.Context, account *model.Account) error
		Get(ctx context.Context, id uuid.UUID) (*model.Account, error)
		Update(ctx context.Context, account *model.Account) error
		Delete(ctx context.Context, id uuid.UUID) error
		List(ctx context.Context) ([]*model.Account, error)
	}

	OrganizationRepository interface {
		CreateOrganization(ctx context.Context, org *model.Organization) error
		GetOrganization(ctx context.Context, id uuid.UUID) (*model.Organization, error)
		UpdateOrganization(ctx context.Context, org *model.Organization) error
		DeleteOrganization(ctx context.Context, id uuid.UUID) error
		ListOrganizations(ctx context.Context, accountID uuid.UUID) ([]*model.Organization, error)
	}

	AppointmentRepository interface {
		Create(ctx context.Context, appointment *model.Appointment) error
		Get(ctx context.Context, id uuid.UUID) (*model.Appointment, error)
		Update(ctx context.Context, appointment *model.Appointment) error
		Delete(ctx context.Context, id uuid.UUID) error
		List(ctx context.Context, filters *model.AppointmentFilters) ([]*model.Appointment, error)
		FindConflictingAppointments(ctx context.Context, staffID uuid.UUID, start, end time.Time) ([]*model.Appointment, error)
		CheckConflicts(ctx context.Context, userID uuid.UUID, startTime, endTime time.Time, excludeID *uuid.UUID) (bool, error)
		GetClinicianAppointments(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*model.Appointment, error)
		GetClinicianSchedule(ctx context.Context, clinicianID uuid.UUID, date time.Time) ([]*model.TimeSlot, error)
	}

	PatientRepository interface {
		Create(ctx context.Context, patient *model.Patient) error
		Get(ctx context.Context, id uuid.UUID) (*model.Patient, error)
		Update(ctx context.Context, patient *model.Patient) error
		Delete(ctx context.Context, id uuid.UUID) error
		List(ctx context.Context, filters *model.PatientFilters) ([]*model.Patient, error)
		DeletePatientAppointments(ctx context.Context, patientID uuid.UUID) error
		AddMedicalRecord(ctx context.Context, record *model.MedicalRecord) error
		GetMedicalRecords(ctx context.Context, patientID uuid.UUID) ([]*model.MedicalRecord, error)
	}

	RBACRepository interface {
		CreateRole(ctx context.Context, role *model.Role) error
		GetRole(ctx context.Context, id uuid.UUID) (*model.Role, error)
		UpdateRole(ctx context.Context, role *model.Role) error
		DeleteRole(ctx context.Context, id uuid.UUID) error
		ListRoles(ctx context.Context, orgID uuid.UUID) ([]*model.Role, error)
		AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID) error
		RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error
		GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*model.Role, error)
		GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*model.Permission, error)
		HasPermission(ctx context.Context, userID uuid.UUID, permission string, organizationID uuid.UUID) (bool, error)
		AddPermissionToRole(ctx context.Context, roleID uuid.UUID, permission string) error
		RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error
		AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error
		AssignRoleToClinician(ctx context.Context, clinicianID, roleID, orgID uuid.UUID) error
		RemoveRoleFromClinician(ctx context.Context, clinicianID, roleID, orgID uuid.UUID) error
		ListRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*model.Permission, error)
		ListClinicianRoles(ctx context.Context, clinicianID, orgID uuid.UUID) ([]*model.Role, error)
		CreatePermission(ctx context.Context, permission *model.Permission) error
		GetPermission(ctx context.Context, id uuid.UUID) (*model.Permission, error)
		UpdatePermission(ctx context.Context, permission *model.Permission) error
		DeletePermission(ctx context.Context, id uuid.UUID) error
		ListPermissions(ctx context.Context) ([]*model.Permission, error)
	}

	AuditRepository interface {
		Create(ctx context.Context, log *model.AuditLog) error
		List(ctx context.Context, filters map[string]interface{}) ([]*model.AuditLog, error)
		ListWithPagination(ctx context.Context, filters map[string]interface{}) ([]*model.AuditLog, int64, error)
		GetAggregateStats(ctx context.Context, filters map[string]interface{}) (*model.AggregateStats, error)
		Cleanup(ctx context.Context, before time.Time) (int64, error)
		DeleteBefore(ctx context.Context, cutoff time.Time) error
	}

	TokenRepository interface {
		StoreVerificationToken(ctx context.Context, userID uuid.UUID, token string, expiry time.Time) error
		ValidateVerificationToken(ctx context.Context, token string) (uuid.UUID, error)
		StoreResetToken(ctx context.Context, userID uuid.UUID, token string, expiry time.Time) error
		ValidateResetToken(ctx context.Context, token string) (uuid.UUID, error)
		InvalidateToken(ctx context.Context, token string) error
		InvalidateVerificationToken(ctx context.Context, token string) error
	}

	RegionRepository interface {
		GetRegion(ctx context.Context, code string) (*model.Region, error)
		ListRegions(ctx context.Context) ([]*model.Region, error)
		GetRegionCodeForCountry(ctx context.Context, countryCode string) (string, error)
		UpdateRegion(ctx context.Context, region *model.Region) error
	}

	ClinicRepository interface {
		Create(ctx context.Context, clinic *model.Clinic) error
		Get(ctx context.Context, id uuid.UUID) (*model.Clinic, error)
		Update(ctx context.Context, clinic *model.Clinic) error
		Delete(ctx context.Context, id uuid.UUID) error
		List(ctx context.Context, organizationID uuid.UUID) ([]*model.Clinic, error)
		AssignStaff(ctx context.Context, staff *model.ClinicStaff) error
		ListStaff(ctx context.Context, clinicID uuid.UUID) ([]*model.ClinicStaff, error)
		RemoveStaff(ctx context.Context, clinicID, userID uuid.UUID) error
		CreateService(ctx context.Context, service *model.Service) error
		GetService(ctx context.Context, serviceID uuid.UUID) (*model.Service, error)
		ListServices(ctx context.Context, clinicID uuid.UUID) ([]*model.Service, error)
		UpdateService(ctx context.Context, service *model.Service) error
		DeleteService(ctx context.Context, serviceID uuid.UUID) error
		DeleteClinicStaff(ctx context.Context, clinicID uuid.UUID) error
	}

	UserRepository interface {
		Create(ctx context.Context, user *model.User) error
		Get(ctx context.Context, id uuid.UUID) (*model.User, error)
		GetByEmail(ctx context.Context, email string) (*model.User, error)
		Update(ctx context.Context, user *model.User) error
		Delete(ctx context.Context, id uuid.UUID) error
		List(ctx context.Context, filters *model.UserFilters) ([]*model.User, error)
		AssignToClinic(ctx context.Context, userID, clinicID uuid.UUID) error
		RemoveFromClinic(ctx context.Context, userID, clinicID uuid.UUID) error
		ListUserClinics(ctx context.Context, userID uuid.UUID) ([]*model.Clinic, error)
		UpdateEmailVerified(ctx context.Context, userID uuid.UUID, verified bool) error
		CreateStaff(ctx context.Context, staff *model.Staff) error
		CreateClinicStaff(ctx context.Context, staff *model.ClinicStaff) error
		GetStaff(ctx context.Context, staffID uuid.UUID) (*model.Staff, error)
	}

	OutboxRepository interface {
		GetPendingEvents(ctx context.Context, limit int) ([]*model.OutboxEvent, error)
		GetPendingEventsWithLock(ctx context.Context, limit int) ([]*model.OutboxEvent, error)
		Create(ctx context.Context, event *model.OutboxEvent) error
		UpdateStatus(ctx context.Context, id uuid.UUID, status model.OutboxStatus, err *string) error
		BeginTx(ctx context.Context) (*sql.Tx, error)
		UpdateStatusTx(ctx context.Context, tx *sql.Tx, id uuid.UUID, status string, errorMessage *string, retryAt *time.Time) error
		MoveToDeadLetter(ctx context.Context, tx *sql.Tx, event *model.OutboxEvent) error
		DeleteProcessedBefore(ctx context.Context, before time.Time) (int64, error)
	}

	MedicalRecordRepository interface {
		Get(ctx context.Context, id uuid.UUID) (*model.MedicalRecord, error)
		List(ctx context.Context, patientID uuid.UUID, filters *model.RecordFilters) ([]*model.MedicalRecord, error)
		CreateWithAudit(ctx context.Context, record *model.MedicalRecord) error
		UpdateWithAudit(ctx context.Context, record *model.MedicalRecord) error
		Delete(ctx context.Context, id uuid.UUID) error
	}

	NotificationRepository interface {
		Create(ctx context.Context, notification *model.Notification) error
		Update(ctx context.Context, notification *model.Notification) error
	}

	ClinicianRepository interface {
		Get(ctx context.Context, id uuid.UUID) (*model.Clinician, error)
	}

	PermissionRepository interface {
		Create(ctx context.Context, permission *model.Permission) error
		Get(ctx context.Context, id uuid.UUID) (*model.Permission, error)
		Update(ctx context.Context, permission *model.Permission) error
		Delete(ctx context.Context, id uuid.UUID) error
		List(ctx context.Context, orgID uuid.UUID) ([]*model.Permission, error)
	}
)
