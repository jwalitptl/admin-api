package patient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
	"github.com/jwalitptl/admin-api/internal/service/audit"

	"github.com/google/uuid"
)

type PatientService interface {
	CreatePatient(ctx context.Context, patient *model.Patient) error
	GetPatient(ctx context.Context, id uuid.UUID) (*model.Patient, error)
	UpdatePatient(ctx context.Context, patient *model.Patient) error
	DeletePatient(ctx context.Context, id uuid.UUID) error
	ListPatients(ctx context.Context, filters *model.PatientFilters) ([]*model.Patient, error)
	CreateAppointment(ctx context.Context, appointment *model.CreateAppointmentRequest) (*model.Appointment, error)
	CancelAppointment(ctx context.Context, appointmentID uuid.UUID, reason string) error
	ListAppointments(ctx context.Context, patientID uuid.UUID, filters *model.AppointmentFilters) ([]*model.Appointment, error)
}

type Service struct {
	repo            repository.PatientRepository
	auditor         *audit.Service
	medicalRepo     repository.MedicalRecordRepository
	appointmentRepo repository.AppointmentRepository
}

func NewService(repo repository.PatientRepository, medicalRepo repository.MedicalRecordRepository, appointmentRepo repository.AppointmentRepository, auditor *audit.Service) *Service {
	return &Service{
		repo:            repo,
		medicalRepo:     medicalRepo,
		appointmentRepo: appointmentRepo,
		auditor:         auditor,
	}
}

func (s *Service) CreatePatient(ctx context.Context, patient *model.Patient) error {
	if err := s.validatePatient(patient); err != nil {
		return fmt.Errorf("invalid patient data: %w", err)
	}

	patient.ID = uuid.New()
	patient.CreatedAt = time.Now()
	patient.UpdatedAt = time.Now()
	patient.Status = string(model.PatientStatusActive)

	// Marshal JSON fields
	if err := s.marshalJSONFields(patient); err != nil {
		return fmt.Errorf("failed to marshal JSON fields: %w", err)
	}

	if err := s.repo.Create(ctx, patient); err != nil {
		return fmt.Errorf("failed to create patient: %w", err)
	}

	s.auditor.Log(ctx, s.getCurrentUserID(ctx), patient.OrganizationID, "create", "patient", patient.ID, &audit.LogOptions{
		Changes: patient,
	})

	return nil
}

func (s *Service) GetPatient(ctx context.Context, id uuid.UUID) (*model.Patient, error) {
	patient, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient: %w", err)
	}

	// Unmarshal JSON fields
	if err := s.unmarshalJSONFields(patient); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON fields: %w", err)
	}

	s.auditor.Log(ctx, s.getCurrentUserID(ctx), patient.OrganizationID, "read", "patient", id, nil)
	return patient, nil
}

func (s *Service) UpdatePatient(ctx context.Context, patient *model.Patient) error {
	if err := s.validatePatient(patient); err != nil {
		return fmt.Errorf("invalid patient data: %w", err)
	}

	patient.UpdatedAt = time.Now()

	// Marshal JSON fields
	if err := s.marshalJSONFields(patient); err != nil {
		return fmt.Errorf("failed to marshal JSON fields: %w", err)
	}

	if err := s.repo.Update(ctx, patient); err != nil {
		return fmt.Errorf("failed to update patient: %w", err)
	}

	s.auditor.Log(ctx, s.getCurrentUserID(ctx), patient.OrganizationID, "update", "patient", patient.ID, &audit.LogOptions{
		Changes: patient,
	})

	return nil
}

func (s *Service) ListPatients(ctx context.Context, filters *model.PatientFilters) ([]*model.Patient, error) {
	patients, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list patients: %w", err)
	}

	// Unmarshal JSON fields for each patient
	for _, patient := range patients {
		if err := s.unmarshalJSONFields(patient); err != nil {
			return nil, fmt.Errorf("failed to unmarshal patient %s: %w", patient.ID, err)
		}
	}

	return patients, nil
}

func (s *Service) DeletePatient(ctx context.Context, id uuid.UUID) error {
	patient, err := s.repo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get patient: %w", err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete patient: %w", err)
	}

	s.auditor.Log(ctx, s.getCurrentUserID(ctx), patient.OrganizationID, "delete", "patient", id, nil)
	return nil
}

func (s *Service) validatePatient(patient *model.Patient) error {
	if patient.ClinicID == uuid.Nil {
		return fmt.Errorf("clinic ID is required")
	}

	if patient.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization ID is required")
	}

	if patient.FirstName == "" {
		return fmt.Errorf("first name is required")
	}

	if patient.LastName == "" {
		return fmt.Errorf("last name is required")
	}

	if patient.Email == "" {
		return fmt.Errorf("email is required")
	}

	if patient.DateOfBirth.IsZero() {
		return fmt.Errorf("date of birth is required")
	}

	return nil
}

func (s *Service) marshalJSONFields(patient *model.Patient) error {
	if patient.EmergencyContact != nil {
		data, err := json.Marshal(patient.EmergencyContact)
		if err != nil {
			return err
		}
		patient.EmergencyContactJSON = string(data)
	}

	if patient.InsuranceInfo != nil {
		data, err := json.Marshal(patient.InsuranceInfo)
		if err != nil {
			return err
		}
		patient.InsuranceInfoJSON = string(data)
	}

	return nil
}

func (s *Service) unmarshalJSONFields(patient *model.Patient) error {
	if patient.EmergencyContactJSON != "" {
		var contact model.EmergencyContact
		if err := json.Unmarshal([]byte(patient.EmergencyContactJSON), &contact); err != nil {
			return err
		}
		patient.EmergencyContact = &contact
	}

	if patient.InsuranceInfoJSON != "" {
		var info model.InsuranceInfo
		if err := json.Unmarshal([]byte(patient.InsuranceInfoJSON), &info); err != nil {
			return err
		}
		patient.InsuranceInfo = &info
	}

	return nil
}

func (s *Service) getCurrentUserID(ctx context.Context) uuid.UUID {
	if ctx == nil {
		return uuid.Nil
	}
	if userID, ok := ctx.Value("user_id").(uuid.UUID); ok {
		return userID
	}
	return uuid.Nil
}

func (s *Service) getPatientOrganizationID(ctx context.Context, patientID uuid.UUID) uuid.UUID {
	patient, err := s.repo.Get(ctx, patientID)
	if err != nil {
		return uuid.Nil
	}
	return patient.OrganizationID
}

func (s *Service) AddMedicalRecord(ctx context.Context, patientID uuid.UUID, record *model.MedicalRecord) error {
	record.ID = uuid.New()
	record.PatientID = patientID
	record.OrganizationID = s.getPatientOrganizationID(ctx, patientID)
	record.CreatedAt = time.Now()
	record.UpdatedAt = time.Now()

	if err := s.repo.AddMedicalRecord(ctx, record); err != nil {
		return fmt.Errorf("failed to add medical record: %w", err)
	}

	// Log audit
	s.auditor.Log(ctx, s.getCurrentUserID(ctx), record.OrganizationID, "create", "medical_record", record.ID, &audit.LogOptions{
		Changes: record,
	})

	return nil
}

func (s *Service) GetMedicalRecord(ctx context.Context, patientID, recordID uuid.UUID) (*model.MedicalRecord, error) {
	records, err := s.repo.GetMedicalRecords(ctx, patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get medical records: %w", err)
	}

	for _, record := range records {
		if record.ID == recordID {
			// Log access
			s.auditor.Log(ctx, s.getCurrentUserID(ctx), record.OrganizationID, "read", "medical_record", recordID, nil)
			return record, nil
		}
	}

	return nil, fmt.Errorf("medical record not found")
}

func (s *Service) ListMedicalRecords(ctx context.Context, patientID uuid.UUID, filters *model.RecordFilters) ([]*model.MedicalRecord, error) {
	records, err := s.repo.GetMedicalRecords(ctx, patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to list medical records: %w", err)
	}

	// Apply filters if provided
	if filters != nil {
		filtered := make([]*model.MedicalRecord, 0)
		for _, record := range records {
			if filters.Type != "" && record.Type != filters.Type {
				continue
			}
			if !filters.StartDate.IsZero() && record.CreatedAt.Before(filters.StartDate) {
				continue
			}
			if !filters.EndDate.IsZero() && record.CreatedAt.After(filters.EndDate) {
				continue
			}
			filtered = append(filtered, record)
		}
		records = filtered
	}

	return records, nil
}

func (s *Service) CreateAppointment(ctx context.Context, req *model.CreateAppointmentRequest) (*model.Appointment, error) {
	appointment := &model.Appointment{
		Base: model.Base{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		ServiceID:   req.ServiceID,
		ClinicID:    uuid.MustParse(req.ClinicID),
		ClinicianID: uuid.MustParse(req.ClinicianID),
		PatientID:   req.PatientID,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Status:      model.AppointmentStatusScheduled,
		Notes:       req.Notes,
	}

	if err := s.appointmentRepo.Create(ctx, appointment); err != nil {
		return nil, fmt.Errorf("failed to create appointment: %w", err)
	}

	// Log audit
	s.auditor.Log(ctx, s.getCurrentUserID(ctx), req.ServiceID, "create", "appointment", appointment.ID, &audit.LogOptions{
		Changes: appointment,
	})

	return appointment, nil
}

func (s *Service) UpdateAppointment(ctx context.Context, appointment *model.Appointment) error {
	appointment.UpdatedAt = time.Now()

	if err := s.appointmentRepo.Update(ctx, appointment); err != nil {
		return fmt.Errorf("failed to update appointment: %w", err)
	}

	// Log audit
	s.auditor.Log(ctx, s.getCurrentUserID(ctx), appointment.ClinicID, "update", "appointment", appointment.ID, &audit.LogOptions{
		Changes: appointment,
	})

	return nil
}

func (s *Service) CancelAppointment(ctx context.Context, appointmentID uuid.UUID, reason string) error {
	appointment, err := s.appointmentRepo.Get(ctx, appointmentID)
	if err != nil {
		return fmt.Errorf("failed to get appointment: %w", err)
	}

	appointment.Status = model.AppointmentStatusCancelled
	appointment.Notes = reason
	appointment.UpdatedAt = time.Now()

	if err := s.appointmentRepo.Update(ctx, appointment); err != nil {
		return fmt.Errorf("failed to cancel appointment: %w", err)
	}

	// Log audit
	s.auditor.Log(ctx, s.getCurrentUserID(ctx), appointment.ClinicID, "cancel", "appointment", appointmentID, &audit.LogOptions{
		Changes: appointment,
	})

	return nil
}

func (s *Service) ListAppointments(ctx context.Context, patientID uuid.UUID, filters *model.AppointmentFilters) ([]*model.Appointment, error) {
	if filters == nil {
		filters = &model.AppointmentFilters{}
	}
	filters.PatientID = patientID
	return s.appointmentRepo.List(ctx, filters)
}
