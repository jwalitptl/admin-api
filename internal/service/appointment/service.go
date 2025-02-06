package appointment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
)

// Add these constants for business rules
const (
	MinAppointmentDuration = 15 * time.Minute
	MaxAppointmentDuration = 4 * time.Hour
	MaxAdvanceBooking      = 90 * 24 * time.Hour // 90 days
	MinAdvanceBooking      = 1 * time.Hour
)

// Extend Repository interface
type Repository interface {
	Create(ctx context.Context, appointment *model.Appointment) error
	Get(ctx context.Context, id uuid.UUID) (*model.Appointment, error)
	List(ctx context.Context, clinicID uuid.UUID, filters map[string]interface{}) ([]*model.Appointment, error)
	Update(ctx context.Context, appointment *model.Appointment) error
	CheckConflicts(ctx context.Context, clinicianID uuid.UUID, startTime, endTime time.Time, excludeID *uuid.UUID) (bool, error)
	GetClinicianAppointments(ctx context.Context, clinicianID uuid.UUID, startDate, endDate time.Time) ([]*model.Appointment, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Add validation function
func (s *Service) validateAppointmentTime(startTime, endTime time.Time) error {
	now := time.Now()
	duration := endTime.Sub(startTime)

	// Check if appointment is in the future
	if startTime.Before(now) {
		return fmt.Errorf("appointment cannot be scheduled in the past")
	}

	// Check minimum advance booking
	if startTime.Sub(now) < MinAdvanceBooking {
		return fmt.Errorf("appointment must be scheduled at least %v in advance", MinAdvanceBooking)
	}

	// Check maximum advance booking
	if startTime.Sub(now) > MaxAdvanceBooking {
		return fmt.Errorf("appointment cannot be scheduled more than %v in advance", MaxAdvanceBooking)
	}

	// Check duration constraints
	if duration < MinAppointmentDuration {
		return fmt.Errorf("appointment duration must be at least %v", MinAppointmentDuration)
	}
	if duration > MaxAppointmentDuration {
		return fmt.Errorf("appointment duration cannot exceed %v", MaxAppointmentDuration)
	}

	return nil
}

func (s *Service) CreateAppointment(ctx context.Context, req *model.CreateAppointmentRequest) (*model.Appointment, error) {
	// Validate time range
	if err := s.validateAppointmentTime(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}

	// Check for scheduling conflicts
	hasConflict, err := s.repo.CheckConflicts(ctx, req.ClinicianID, req.StartTime, req.EndTime, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check scheduling conflicts: %w", err)
	}
	if hasConflict {
		return nil, fmt.Errorf("scheduling conflict: clinician already has an appointment during this time")
	}

	appointment := &model.Appointment{
		Base: model.Base{
			ID: uuid.New(),
		},
		ClinicID:    req.ClinicID,
		ClinicianID: req.ClinicianID,
		PatientID:   req.PatientID,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Status:      model.AppointmentStatusScheduled,
		Notes:       req.Notes,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, appointment); err != nil {
		return nil, fmt.Errorf("failed to create appointment: %w", err)
	}

	return appointment, nil
}

// Add method to check clinician availability
func (s *Service) GetClinicianAvailability(ctx context.Context, clinicianID uuid.UUID, date time.Time) ([]model.TimeSlot, error) {
	// Get start and end of business hours (9 AM to 5 PM)
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, date.Location())
	endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 17, 0, 0, 0, date.Location())

	// Get all appointments for the day
	appointments, err := s.repo.GetClinicianAppointments(ctx, clinicianID, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("failed to get clinician appointments: %w", err)
	}

	// Generate available time slots
	slots := generateTimeSlots(startOfDay, endOfDay, 30*time.Minute) // 30-minute slots
	availableSlots := filterAvailableSlots(slots, appointments)

	return availableSlots, nil
}

func (s *Service) UpdateAppointment(ctx context.Context, id uuid.UUID, req *model.UpdateAppointmentRequest) (*model.Appointment, error) {
	appointment, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointment: %w", err)
	}

	// Store original times for conflict checking
	originalStart := appointment.StartTime
	originalEnd := appointment.EndTime

	// Update fields if provided
	if req.StartTime != nil {
		appointment.StartTime = *req.StartTime
	}
	if req.EndTime != nil {
		appointment.EndTime = *req.EndTime
	}
	if req.Status != nil {
		// Validate status transition
		if err := validateStatusTransition(appointment.Status, *req.Status); err != nil {
			return nil, err
		}
		appointment.Status = *req.Status
	}
	if req.Notes != nil {
		appointment.Notes = *req.Notes
	}
	if req.CancelReason != nil {
		appointment.CancelReason = *req.CancelReason
	}

	// If times were updated, validate and check conflicts
	if req.StartTime != nil || req.EndTime != nil {
		if err := s.validateAppointmentTime(appointment.StartTime, appointment.EndTime); err != nil {
			return nil, err
		}

		// Only check conflicts if the time has changed
		if !appointment.StartTime.Equal(originalStart) || !appointment.EndTime.Equal(originalEnd) {
			hasConflict, err := s.repo.CheckConflicts(ctx, appointment.ClinicianID, appointment.StartTime, appointment.EndTime, &id)
			if err != nil {
				return nil, fmt.Errorf("failed to check scheduling conflicts: %w", err)
			}
			if hasConflict {
				return nil, fmt.Errorf("scheduling conflict: clinician already has an appointment during this time")
			}
		}
	}

	appointment.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, appointment); err != nil {
		return nil, fmt.Errorf("failed to update appointment: %w", err)
	}

	return appointment, nil
}

// Helper function to validate status transitions
func validateStatusTransition(current, new model.AppointmentStatus) error {
	// Define valid transitions
	validTransitions := map[model.AppointmentStatus][]model.AppointmentStatus{
		model.AppointmentStatusScheduled: {
			model.AppointmentStatusConfirmed,
			model.AppointmentStatusCancelled,
		},
		model.AppointmentStatusConfirmed: {
			model.AppointmentStatusCompleted,
			model.AppointmentStatusCancelled,
		},
		model.AppointmentStatusCancelled: {}, // No valid transitions from cancelled
		model.AppointmentStatusCompleted: {}, // No valid transitions from completed
	}

	allowed := validTransitions[current]
	for _, status := range allowed {
		if status == new {
			return nil
		}
	}

	return fmt.Errorf("invalid status transition from %s to %s", current, new)
}

// Add these methods back
func (s *Service) GetAppointment(ctx context.Context, id uuid.UUID) (*model.Appointment, error) {
	appointment, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointment: %w", err)
	}
	return appointment, nil
}

func (s *Service) ListAppointments(ctx context.Context, clinicID uuid.UUID, filters map[string]interface{}) ([]*model.Appointment, error) {
	appointments, err := s.repo.List(ctx, clinicID, filters)
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
