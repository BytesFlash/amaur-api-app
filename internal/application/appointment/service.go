package appointment

import (
	"context"
	"errors"
	"time"

	"amaur/api/internal/domain/appointment"
	"amaur/api/internal/domain/program"
	"amaur/api/internal/domain/worker"
	"amaur/api/pkg/timeutil"

	"github.com/google/uuid"
)

var (
	ErrNotFound            = errors.New("appointment not found")
	ErrInvalidDate         = errors.New("invalid scheduled_at date/time")
	ErrInvalidRecurrence   = errors.New("session_count must be between 1 and 52")
	ErrInvalidStatus       = errors.New("invalid appointment status")
	ErrOutsideAvailability = errors.New("la hora seleccionada está fuera de los bloques de disponibilidad del profesional")
	ErrTooSoon             = errors.New("solo se pueden agendar horarios desde la próxima hora disponible")
	ErrWorkerBusy          = errors.New("worker already has another booking in this time block")
)

type CreateAppointmentRequest struct {
	PatientID        uuid.UUID  `json:"patient_id"`
	WorkerID         *uuid.UUID `json:"worker_id"`
	ServiceTypeID    uuid.UUID  `json:"service_type_id"`
	CompanyID        *uuid.UUID `json:"company_id"`
	ScheduledAt      string     `json:"scheduled_at"` // RFC3339 or "YYYY-MM-DDTHH:MM"
	DurationMinutes  *int       `json:"duration_minutes"`
	Notes            *string    `json:"notes"`
	ChiefComplaint   *string    `json:"chief_complaint"`
	Subjective       *string    `json:"subjective"`
	Objective        *string    `json:"objective"`
	Assessment       *string    `json:"assessment"`
	Plan             *string    `json:"plan"`
	FollowUpRequired *bool      `json:"follow_up_required"`
	FollowUpNotes    *string    `json:"follow_up_notes"`
	FollowUpDate     *string    `json:"follow_up_date"` // YYYY-MM-DD
	// Recurring booking
	SessionCount   int `json:"session_count"`   // 1 = single, >1 = batch
	FrequencyWeeks int `json:"frequency_weeks"` // 1=weekly, 2=biweekly
}

type UpdateAppointmentRequest struct {
	WorkerID         *uuid.UUID `json:"worker_id"`
	ServiceTypeID    *uuid.UUID `json:"service_type_id"`
	ScheduledAt      *string    `json:"scheduled_at"`
	DurationMinutes  *int       `json:"duration_minutes"`
	Status           *string    `json:"status"`
	Notes            *string    `json:"notes"`
	ChiefComplaint   *string    `json:"chief_complaint"`
	Subjective       *string    `json:"subjective"`
	Objective        *string    `json:"objective"`
	Assessment       *string    `json:"assessment"`
	Plan             *string    `json:"plan"`
	FollowUpRequired *bool      `json:"follow_up_required"`
	FollowUpNotes    *string    `json:"follow_up_notes"`
	FollowUpDate     *string    `json:"follow_up_date"` // YYYY-MM-DD
}

type Service struct {
	repo        appointment.Repository
	programRepo program.Repository
	workerRepo  worker.Repository
}

func NewService(repo appointment.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) WithProgramRepo(r program.Repository) *Service {
	s.programRepo = r
	return s
}

func (s *Service) WithWorkerRepo(r worker.Repository) *Service {
	s.workerRepo = r
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
			if err := s.ensureWorkerSlotAllowed(ctx, *req.WorkerID, scheduledAt, duration); err != nil {
				return nil, err
			}
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
			Status:           appointment.StatusRequested,
			Notes:            req.Notes,
			ChiefComplaint:   req.ChiefComplaint,
			Subjective:       req.Subjective,
			Objective:        req.Objective,
			Assessment:       req.Assessment,
			Plan:             req.Plan,
			FollowUpRequired: coalesceBool(req.FollowUpRequired),
			FollowUpNotes:    req.FollowUpNotes,
			FollowUpDate:     parseDateOnlyPtr(req.FollowUpDate),
			CreatedBy:        &by,
		})
	}

	if err := s.repo.CreateBatch(ctx, batch); err != nil {
		return nil, err
	}

	created := make([]*appointment.Appointment, 0, len(batch))
	for _, item := range batch {
		enriched, err := s.repo.FindByID(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		created = append(created, enriched)
	}

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*appointment.Appointment, error) {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	_ = s.autoCompleteIfExpired(ctx, a)
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
		if !isValidStatus(*req.Status) {
			return nil, ErrInvalidStatus
		}
		a.Status = *req.Status
	}
	if req.Notes != nil {
		a.Notes = req.Notes
	}
	if req.ChiefComplaint != nil {
		a.ChiefComplaint = req.ChiefComplaint
	}
	if req.Subjective != nil {
		a.Subjective = req.Subjective
	}
	if req.Objective != nil {
		a.Objective = req.Objective
	}
	if req.Assessment != nil {
		a.Assessment = req.Assessment
	}
	if req.Plan != nil {
		a.Plan = req.Plan
	}
	if req.FollowUpRequired != nil {
		a.FollowUpRequired = *req.FollowUpRequired
	}
	if req.FollowUpNotes != nil {
		a.FollowUpNotes = req.FollowUpNotes
	}
	if req.FollowUpDate != nil {
		a.FollowUpDate = parseDateOnlyPtr(req.FollowUpDate)
	}
	if a.WorkerID != nil {
		duration := coalesceDuration(a.DurationMinutes)
		if err := s.ensureWorkerSlotAllowed(ctx, *a.WorkerID, a.ScheduledAt, duration); err != nil {
			return nil, err
		}
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
	items, total, err := s.repo.List(ctx, f, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	for _, item := range items {
		_ = s.autoCompleteIfExpired(ctx, item)
	}
	return items, total, nil
}

func parseScheduledAt(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.In(timeutil.Santiago()), nil
	}
	for _, f := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(f, s, timeutil.Santiago()); err == nil {
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

func isValidStatus(status string) bool {
	switch status {
	case appointment.StatusRequested,
		appointment.StatusConfirmed,
		appointment.StatusInProgress,
		appointment.StatusCompleted,
		appointment.StatusCancelled,
		appointment.StatusNoShow:
		return true
	default:
		return false
	}
}

func parseDateOnlyPtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	if t, err := time.Parse("2006-01-02", *s); err == nil {
		return &t
	}
	return nil
}

func coalesceBool(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func (s *Service) autoCompleteIfExpired(ctx context.Context, item *appointment.Appointment) error {
	if item == nil || item.Status != appointment.StatusInProgress {
		return nil
	}
	endAt := item.ScheduledAt.Add(time.Duration(coalesceDuration(item.DurationMinutes)) * time.Minute)
	if endAt.After(time.Now().In(timeutil.Santiago())) {
		return nil
	}
	item.Status = appointment.StatusCompleted
	now := time.Now()
	item.UpdatedAt = &now
	return s.repo.Update(ctx, item)
}

func (s *Service) ensureWorkerSlotAllowed(ctx context.Context, workerID uuid.UUID, scheduledAt time.Time, duration int) error {
	if scheduledAt.Before(nextBookableHour()) {
		return ErrTooSoon
	}
	if s.workerRepo == nil {
		return nil
	}
	rules, err := s.workerRepo.ListAvailabilityRules(ctx, workerID)
	if err != nil {
		return err
	}
	if isWithinAvailability(rules, scheduledAt, duration) {
		return nil
	}
	return ErrOutsideAvailability
}

func nextBookableHour() time.Time {
	now := time.Now().In(timeutil.Santiago())
	return now.Truncate(time.Hour).Add(time.Hour)
}

func isWithinAvailability(rules []*worker.AvailabilityRule, scheduledAt time.Time, duration int) bool {
	if duration <= 0 {
		duration = 60
	}
	slotEnd := scheduledAt.Add(time.Duration(duration) * time.Minute)
	for _, rule := range rules {
		if rule == nil || !rule.IsActive || rule.Weekday != int16(scheduledAt.Weekday()) {
			continue
		}
		ruleStartTime, err := time.ParseInLocation("15:04", rule.StartTime, scheduledAt.Location())
		if err != nil {
			continue
		}
		ruleEndTime, err := time.ParseInLocation("15:04", rule.EndTime, scheduledAt.Location())
		if err != nil {
			continue
		}
		ruleStart := time.Date(
			scheduledAt.Year(),
			scheduledAt.Month(),
			scheduledAt.Day(),
			ruleStartTime.Hour(),
			ruleStartTime.Minute(),
			0,
			0,
			scheduledAt.Location(),
		)
		ruleEnd := time.Date(
			scheduledAt.Year(),
			scheduledAt.Month(),
			scheduledAt.Day(),
			ruleEndTime.Hour(),
			ruleEndTime.Minute(),
			0,
			0,
			scheduledAt.Location(),
		)
		if scheduledAt.Before(ruleStart) || slotEnd.After(ruleEnd) {
			continue
		}
		if int(scheduledAt.Sub(ruleStart).Minutes())%duration != 0 {
			continue
		}
		return true
	}
	return false
}
