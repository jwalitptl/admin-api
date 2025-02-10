package appointment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
	"github.com/jwalitptl/admin-api/internal/service/audit"
	"github.com/jwalitptl/admin-api/internal/service/notification"
)

// Add these constants for business rules
const (
	MinAppointmentDuration = 15 * time.Minute
	MaxAppointmentDuration = 4 * time.Hour
	MaxAdvanceBooking      = 90 * 24 * time.Hour // 90 days
	MinAdvanceBooking      = 1 * time.Hour
)

type Service struct {
	repo         repository.AppointmentRepository
	notifSvc     notification.Service
	auditor      *audit.Service
	clinicianSvc repository.ClinicianRepository
}

func NewService(repo repository.AppointmentRepository, notifSvc notification.Service, clinicianSvc repository.ClinicianRepository, auditor *audit.Service) *Service {
	return &Service{
		repo:         repo,
		notifSvc:     notifSvc,
		clinicianSvc: clinicianSvc,
		auditor:      auditor,
	}
}

// Add validation function
func (s *Service) validateAppointmentTime(startTime, endTime time.Time) error {
	now := time.Now()
	duration := endTime.Sub(startTime)

	if startTime.Before(now) {
		return fmt.Errorf("appointment cannot be scheduled in the past")
	}

	if duration < MinAppointmentDuration {
		return fmt.Errorf("appointment duration must be at least %v", MinAppointmentDuration)
	}

	if duration > MaxAppointmentDuration {
		return fmt.Errorf("appointment duration cannot exceed %v", MaxAppointmentDuration)
	}

	return nil
}

func (s *Service) CreateAppointment(ctx context.Context, apt *model.Appointment) error {
	if err := s.validateAppointment(apt); err != nil {
		return fmt.Errorf("invalid appointment: %w", err)
	}

	apt.ID = uuid.New()
	apt.Status = model.AppointmentStatusScheduled
	apt.CreatedAt = time.Now()
	apt.UpdatedAt = time.Now()

	if err := s.repo.Create(ctx, apt); err != nil {
		return fmt.Errorf("failed to create appointment: %w", err)
	}

	// Send notifications
	if err := s.notifyParticipants(ctx, apt, "appointment_created"); err != nil {
		s.auditor.Log(ctx, apt.PatientID, apt.ClinicID, "notification_failed", "appointment", apt.ID, &audit.LogOptions{
			Metadata: map[string]interface{}{
				"error": err.Error(),
			},
		})
	}

	s.auditor.Log(ctx, apt.PatientID, apt.ClinicID, "create", "appointment", apt.ID, &audit.LogOptions{
		Changes: apt,
	})

	return nil
}

func (s *Service) GetAppointment(ctx context.Context, id uuid.UUID) (*model.Appointment, error) {
	apt, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointment: %w", err)
	}

	s.auditor.Log(ctx, apt.PatientID, apt.ClinicID, "read", "appointment", id, nil)
	return apt, nil
}

func (s *Service) UpdateAppointment(ctx context.Context, apt *model.Appointment) error {
	if err := s.validateAppointment(apt); err != nil {
		return fmt.Errorf("invalid appointment: %w", err)
	}

	apt.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, apt); err != nil {
		return fmt.Errorf("failed to update appointment: %w", err)
	}

	// Send notifications
	if err := s.notifyParticipants(ctx, apt, "appointment_updated"); err != nil {
		s.auditor.Log(ctx, apt.PatientID, apt.ClinicID, "notification_failed", "appointment", apt.ID, &audit.LogOptions{
			Metadata: map[string]interface{}{
				"error": err.Error(),
			},
		})
	}

	s.auditor.Log(ctx, apt.PatientID, apt.ClinicID, "update", "appointment", apt.ID, &audit.LogOptions{
		Changes: apt,
	})

	return nil
}

func (s *Service) CancelAppointment(ctx context.Context, id uuid.UUID, reason string) error {
	apt, err := s.repo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get appointment: %w", err)
	}

	if apt.Status == model.AppointmentStatusCancelled {
		return fmt.Errorf("appointment is already cancelled")
	}

	if apt.Status == model.AppointmentStatusCompleted {
		return fmt.Errorf("cannot cancel a completed appointment")
	}

	apt.Status = model.AppointmentStatusCancelled
	apt.CancelReason = &reason
	apt.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, apt); err != nil {
		return fmt.Errorf("failed to cancel appointment: %w", err)
	}

	// Send notifications
	if err := s.notifyParticipants(ctx, apt, "appointment_cancelled"); err != nil {
		s.auditor.Log(ctx, apt.PatientID, apt.ClinicID, "notification_failed", "appointment", apt.ID, &audit.LogOptions{
			Metadata: map[string]interface{}{
				"error": err.Error(),
			},
		})
	}

	s.auditor.Log(ctx, apt.PatientID, apt.ClinicID, "cancel", "appointment", id, &audit.LogOptions{
		Changes: map[string]interface{}{
			"status":        apt.Status,
			"cancel_reason": reason,
		},
	})

	return nil
}

func (s *Service) isTimeSlotAvailable(ctx context.Context, staffID uuid.UUID, start, end time.Time) (bool, error) {
	conflicts, err := s.repo.FindConflictingAppointments(ctx, staffID, start, end)
	if err != nil {
		return false, err
	}
	return len(conflicts) == 0, nil
}

func (s *Service) GetClinicianAvailability(ctx context.Context, clinicianID uuid.UUID, date time.Time) ([]model.TimeSlot, error) {
	schedule, err := s.repo.GetClinicianSchedule(ctx, clinicianID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get clinician schedule: %w", err)
	}

	appointments, err := s.repo.GetClinicianAppointments(ctx, clinicianID, date, date.Add(24*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("failed to get clinician appointments: %w", err)
	}

	return s.calculateAvailableSlots(schedule, appointments), nil
}

func (s *Service) ListAppointments(ctx context.Context, filters *model.AppointmentFilters) ([]*model.Appointment, error) {
	appointments, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list appointments: %w", err)
	}
	return appointments, nil
}

// Add these types and functions
func generateTimeSlots(start, end time.Time, duration time.Duration) []model.TimeSlot {
	var slots []model.TimeSlot
	for t := start; t.Before(end); t = t.Add(duration) {
		slots = append(slots, model.TimeSlot{
			Start: t,
			End:   t.Add(duration),
		})
	}
	return slots
}

func filterAvailableSlots(slots []model.TimeSlot, appointments []*model.Appointment) []model.TimeSlot {
	var available []model.TimeSlot
	for _, slot := range slots {
		conflict := false
		for _, apt := range appointments {
			if !(slot.End.Before(apt.StartTime) || slot.Start.After(apt.EndTime)) {
				conflict = true
				break
			}
		}
		if !conflict {
			available = append(available, slot)
		}
	}
	return available
}

func (s *Service) DeleteAppointment(ctx context.Context, id uuid.UUID) error {
	// Only allow deletion of cancelled appointments
	appointment, err := s.repo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get appointment: %w", err)
	}

	if appointment.Status != model.AppointmentStatusCancelled {
		return fmt.Errorf("can only delete cancelled appointments")
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) validateAppointment(apt *model.Appointment) error {
	if apt.PatientID == uuid.Nil {
		return fmt.Errorf("patient ID is required")
	}

	if apt.ClinicianID == uuid.Nil {
		return fmt.Errorf("clinician ID is required")
	}

	if apt.ClinicID == uuid.Nil {
		return fmt.Errorf("clinic ID is required")
	}

	duration := apt.EndTime.Sub(apt.StartTime)
	if duration < MinAppointmentDuration || duration > MaxAppointmentDuration {
		return fmt.Errorf("invalid appointment duration: must be between %v and %v", MinAppointmentDuration, MaxAppointmentDuration)
	}

	advance := apt.StartTime.Sub(time.Now())
	if advance < MinAdvanceBooking || advance > MaxAdvanceBooking {
		return fmt.Errorf("invalid booking time: must be between %v and %v in advance", MinAdvanceBooking, MaxAdvanceBooking)
	}

	hasConflict, err := s.repo.CheckConflict(context.Background(), apt)
	if err != nil {
		return fmt.Errorf("failed to check conflicts: %w", err)
	}
	if hasConflict {
		return fmt.Errorf("appointment conflicts with existing booking")
	}

	return nil
}

func (s *Service) GetAvailableSlots(ctx context.Context, clinicianID uuid.UUID, date time.Time) ([]model.TimeSlot, error) {
	clinician, err := s.clinicianSvc.Get(ctx, clinicianID)
	if err != nil {
		return nil, fmt.Errorf("failed to get clinician: %w", err)
	}

	schedule := s.getClinicianSchedule(clinician, date)
	appointments, err := s.repo.GetClinicianAppointments(ctx, clinicianID, date, date.Add(24*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("failed to get clinician appointments: %w", err)
	}

	return s.calculateAvailableSlots(schedule, appointments), nil
}

func (s *Service) notifyParticipants(ctx context.Context, apt *model.Appointment, event string) error {
	// Implementation of notifyParticipants method
	return nil
}

func (s *Service) getClinicianSchedule(clinician *model.Clinician, date time.Time) []*model.TimeSlot {
	// Implementation of getClinicianSchedule method
	return nil
}

func (s *Service) calculateAvailableSlots(schedule []*model.TimeSlot, appointments []*model.Appointment) []model.TimeSlot {
	// Implementation of calculateAvailableSlots method
	return nil
}
