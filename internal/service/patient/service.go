package patient

import (
	"context"

	"github.com/jwalitptl/admin-api/internal/model"

	"github.com/google/uuid"
)

type PatientService interface {
	CreatePatient(ctx context.Context, patient *model.Patient) error
	GetPatient(ctx context.Context, id uuid.UUID) (*model.Patient, error)
	ListPatients(ctx context.Context, clinicID uuid.UUID) ([]*model.Patient, error)
	DeletePatient(ctx context.Context, id uuid.UUID) error
	UpdatePatient(ctx context.Context, id uuid.UUID, req *model.UpdatePatientRequest) (*model.Patient, error)
}

type Repository interface {
	Create(ctx context.Context, patient *model.Patient) error
	Get(ctx context.Context, id uuid.UUID) (*model.Patient, error)
	List(ctx context.Context, clinicID uuid.UUID) ([]*model.Patient, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Update(ctx context.Context, patient *model.Patient) error
	DeletePatientAppointments(ctx context.Context, patientID uuid.UUID) error
}

type serviceImpl struct {
	repo Repository
}

func NewService(repo Repository) PatientService {
	return &serviceImpl{repo: repo}
}

func (s *serviceImpl) CreatePatient(ctx context.Context, patient *model.Patient) error {
	patient.ID = uuid.New()
	return s.repo.Create(ctx, patient)
}

func (s *serviceImpl) GetPatient(ctx context.Context, id uuid.UUID) (*model.Patient, error) {
	return s.repo.Get(ctx, id)
}

func (s *serviceImpl) ListPatients(ctx context.Context, clinicID uuid.UUID) ([]*model.Patient, error) {
	return s.repo.List(ctx, clinicID)
}

func (s *serviceImpl) DeletePatient(ctx context.Context, id uuid.UUID) error {
	// First delete all appointments for this patient
	if err := s.repo.DeletePatientAppointments(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *serviceImpl) UpdatePatient(ctx context.Context, id uuid.UUID, req *model.UpdatePatientRequest) (*model.Patient, error) {
	patient, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields from request
	if req.FirstName != nil {
		patient.Name = *req.FirstName
	}
	if req.Email != nil {
		patient.Email = *req.Email
	}
	if req.Status != nil {
		patient.Status = *req.Status
	}

	err = s.repo.Update(ctx, patient)
	if err != nil {
		return nil, err
	}
	return patient, nil
}
