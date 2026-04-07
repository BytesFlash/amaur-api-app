package appointment

import (
	"context"
	"errors"
	"time"

	"amaur/api/internal/domain/appointment"
	"amaur/api/internal/domain/program"

	"github.com/google/uuid"
)

var (
	ErrNotFound          = errors.New("appointment not found")
	ErrInvalidDate       = errors.New("invalid scheduled_at date/time")
	ErrInvalidRecurrence = errors.New("session_count must be between 1 and 52")
	ErrWorkerBusy        = errors.New("worker already has another booking in this time block")
)

type CreateAppointmentRequest struct {
	PatientID       uuid.UUID  `json:"patient_id"`
	WorkerID        *uuid.UUID `json:"worker_id"`
	ServiceTypeID   uuid.UUID  `json:"service_type_id"`
	CompanyID       *uuid.UUID `json:"company_id"`
	ScheduledAt     string     `json:"scheduled_at"` // RFC3339 or "YYYY-MM-DDTHH:MM"
	DurationMinutes *int       `json:"duration_minutes"`
	Notes           *string    `json:"notes"`
	// Recurring booking
	SessionCount   int `json:"session_count"`   // 1 = single, >1 = batch
	FrequencyWeeks int `json:"frequency_weeks"` // 1=weekly, 2=biweekly
}

type UpdateAppointmentRequest struct {
	WorkerID        *uuid.UUID `json:"worker_id"`
	ServiceTypeID   *uuid.UUID `json:"service_type_id"`
	ScheduledAt     *string    `json:"scheduled_at"`
	DurationMinutes *int       `json:"duration_minutes"`
	Status          *string    `json:"status"`
	Notes           *string    `json:"notes"`
}

type Service struct {
	repo        appointment.Repository
	programRepo program.Repository
}

func NewService(repo appointment.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) WithProgramRepo(r program.Repository) *Service {
	s.programRepo = r
	return s
}

// Create creates one or more appointments (recurring if SessionCount > 1).
func (s *Service) Create(ctx context.Context, req CreateAppointmentRequest, by uuid.UUID) ([]*appointment.Appointment, error) {
	first, err := parseScheduledAt(req.ScheduledAt)
	if err != nil {
		return nil, ErrInvalidDate
	}

	count := req.SessionCount
	if count < 1 {
		count = 1
	}
	if count > 52 {
		return nil, ErrInvalidRecurrence
	}

	freqWeeks := req.FrequencyWeeks
	if freqWeeks < 1 {
		freqWeeks = 1
	}

	var groupID *uuid.UUID
	if count > 1 {
		g := uuid.New()
		groupID = &g
	}

	batch := make([]*appointment.Appointment, 0, count)
	for i := 0; i < count; i++ {
		scheduledAt := first.AddDate(0, 0, i*freqWeeks*7)
		if req.WorkerID != nil {
			duration := coalesceDuration(req.DurationMinutes)
			occupied, err := s.repo.HasWorkerConflict(ctx, *req.WorkerID, scheduledAt, duration, nil)
			if err != nil {
				return nil, err
			}
			if occupied {
				return nil, ErrWorkerBusy
			}
			if s.programRepo != nil {
				groupOccupied, err := s.programRepo.HasWorkerScheduleConflict(
					ctx,
					*req.WorkerID,
					scheduledAt,
					scheduledAt.Format("15:04"),
					duration,
					nil,
				)
				if err != nil {
					return nil, err
				}
				if groupOccupied {
					return nil, ErrWorkerBusy
				}
			}
		}
		batch = append(batch, &appointment.Appointment{
			ID:               uuid.New(),
			PatientID:        req.PatientID,
			WorkerID:         req.WorkerID,
			ServiceTypeID:    req.ServiceTypeID,
			CompanyID:        req.CompanyID,
			RecurringGroupID: groupID,
			ScheduledAt:      scheduledAt,
			DurationMinutes:  req.DurationMinutes,
			Status:           "confirmed",
			Notes:            req.Notes,
			CreatedBy:        &by,
		})
	}

	if err := s.repo.CreateBatch(ctx, batch); err != nil {
		return nil, err
	}
	return batch, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*appointment.Appointment, error) {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return a, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateAppointmentRequest, by uuid.UUID) (*appointment.Appointment, error) {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	now := time.Now()
	if req.WorkerID != nil {
		a.WorkerID = req.WorkerID
	}
	if req.ServiceTypeID != nil {
		a.ServiceTypeID = *req.ServiceTypeID
	}
	if req.ScheduledAt != nil {
		t, err := parseScheduledAt(*req.ScheduledAt)
		if err != nil {
			return nil, ErrInvalidDate
		}
		a.ScheduledAt = t
	}
	if req.DurationMinutes != nil {
		a.DurationMinutes = req.DurationMinutes
	}
	if req.Status != nil {
		a.Status = *req.Status
	}
	if req.Notes != nil {
		a.Notes = req.Notes
	}
	if a.WorkerID != nil {
		duration := coalesceDuration(a.DurationMinutes)
		occupied, err := s.repo.HasWorkerConflict(ctx, *a.WorkerID, a.ScheduledAt, duration, &a.ID)
		if err != nil {
			return nil, err
		}
		if occupied {
			return nil, ErrWorkerBusy
		}
		if s.programRepo != nil {
			groupOccupied, err := s.programRepo.HasWorkerScheduleConflict(
				ctx,
				*a.WorkerID,
				a.ScheduledAt,
				a.ScheduledAt.Format("15:04"),
				duration,
				nil,
			)
			if err != nil {
				return nil, err
			}
			if groupOccupied {
				return nil, ErrWorkerBusy
			}
		}
	}
	a.UpdatedAt = &now
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return ErrNotFound
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context, patientID, workerID, companyID, status, dateFrom, dateTo string, limit, offset int) ([]*appointment.Appointment, int64, error) {
	f := appointment.Filter{Status: status}
	if patientID != "" {
		if id, err := uuid.Parse(patientID); err == nil {
			f.PatientID = &id
		}
	}
	if workerID != "" {
		if id, err := uuid.Parse(workerID); err == nil {
			f.WorkerID = &id
		}
	}
	if companyID != "" {
		if id, err := uuid.Parse(companyID); err == nil {
			f.CompanyID = &id
		}
	}
	if dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			f.DateFrom = &t
		}
	}
	if dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			f.DateTo = &t
		}
	}
	return s.repo.List(ctx, f, limit, offset)
}

func parseScheduledAt(s string) (time.Time, error) {
	formats := []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02T15:04:05", "2006-01-02"}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid date format")
}

func coalesceDuration(duration *int) int {
	if duration != nil && *duration > 0 {
		return *duration
	}
	return 60
}
