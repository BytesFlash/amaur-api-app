package treatmentplan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"amaur/api/internal/domain/appointment"
	"amaur/api/internal/domain/treatmentplan"
	"amaur/api/internal/domain/worker"
	"amaur/api/pkg/timeutil"

	"github.com/google/uuid"
)

var (
	ErrNotFound            = errors.New("treatment plan not found")
	ErrInvalidStatus       = errors.New("invalid treatment plan status")
	ErrPlanTerminal        = errors.New("treatment plan is already completed or cancelled")
	ErrNothingToGenerate   = errors.New("all sessions are already scheduled or completed")
	ErrSlotUnavailable     = errors.New("one or more slots are outside the professional's availability")
	ErrSlotConflict        = errors.New("one or more slots conflict with existing appointments")
)

// ── Requests ──────────────────────────────────────────────────────────────────

type CreateRequest struct {
	PatientID         uuid.UUID  `json:"patient_id"`
	ProfessionalID    *uuid.UUID `json:"professional_id"`
	ServiceTypeID     uuid.UUID  `json:"service_type_id"`
	Title             string     `json:"title"`
	Objective         *string    `json:"objective"`
	TotalSessions     int        `json:"total_sessions"`
	FrequencyType     string     `json:"frequency_type"`
	FrequencyInterval *int       `json:"frequency_interval"` // nil → derivado de FrequencyType
	StartDate         string     `json:"start_date"`         // YYYY-MM-DD
	Notes             *string    `json:"notes"`
}

type UpdateRequest struct {
	ProfessionalID    *uuid.UUID `json:"professional_id"`
	ServiceTypeID     *uuid.UUID `json:"service_type_id"`
	Title             *string    `json:"title"`
	Objective         *string    `json:"objective"`
	TotalSessions     *int       `json:"total_sessions"`
	FrequencyType     *string    `json:"frequency_type"`
	FrequencyInterval *int       `json:"frequency_interval"`
	StartDate         *string    `json:"start_date"`
	Notes             *string    `json:"notes"`
}

// SlotInput is a single explicitly-provided session date used in confirmed generate requests.
type SlotInput struct {
	SessionNumber int    `json:"session_number"`
	ScheduledAt   string `json:"scheduled_at"` // RFC3339 or YYYY-MM-DDTHH:MM
}

// GenerateSessionsRequest allows creating recurring appointments for the plan.
// Either StartAt (auto-compute from frequency) or Slots (explicit dates) must be provided.
type GenerateSessionsRequest struct {
	WorkerID        *uuid.UUID  `json:"worker_id"`
	DurationMinutes *int        `json:"duration_minutes"`
	// Auto mode: derive all N dates from start + frequency interval
	StartAt string `json:"start_at,omitempty"` // RFC3339 or YYYY-MM-DDTHH:MM
	// Confirmed mode: user-adjusted explicit slots (takes precedence over StartAt)
	Slots []SlotInput `json:"slots,omitempty"`
}

// PreviewRequest is the input for a dry-run preview (no appointments created).
type PreviewRequest struct {
	WorkerID        *uuid.UUID `json:"worker_id"`
	DurationMinutes *int       `json:"duration_minutes"`
	StartAt         string     `json:"start_at"` // RFC3339 or YYYY-MM-DDTHH:MM
}

// ── Response types ────────────────────────────────────────────────────────────

// SessionSlotPreview describes a single proposed session in a preview response.
type SessionSlotPreview struct {
	SessionNumber  int    `json:"session_number"`
	ScheduledAt    string `json:"scheduled_at"` // RFC3339 in Santiago TZ
	Available      bool   `json:"available"`    // within worker availability rules
	ConflictReason string `json:"conflict_reason,omitempty"` // human-readable if !Available
}

// PreviewResult is the response body of the preview endpoint.
type PreviewResult struct {
	TotalToCreate int                  `json:"total_to_create"`
	AllAvailable  bool                 `json:"all_available"`
	Slots         []SessionSlotPreview `json:"slots"`
}

// ── Service ───────────────────────────────────────────────────────────────────

type Service struct {
	repo       treatmentplan.Repository
	apptRepo   appointment.Repository
	workerRepo worker.Repository // optional; enables availability validation
}

func NewService(repo treatmentplan.Repository, apptRepo appointment.Repository) *Service {
	return &Service{repo: repo, apptRepo: apptRepo}
}

func (s *Service) WithWorkerRepo(r worker.Repository) *Service {
	s.workerRepo = r
	return s
}

// ── CRUD ──────────────────────────────────────────────────────────────────────

func (s *Service) Create(ctx context.Context, req CreateRequest, by uuid.UUID) (*treatmentplan.TreatmentPlan, error) {
	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, errors.New("invalid start_date, expected YYYY-MM-DD")
	}
	freqType := req.FrequencyType
	if freqType == "" {
		freqType = treatmentplan.FrequencyWeekly
	}
	interval := treatmentplan.FrequencyIntervalDays(freqType, 0)
	if req.FrequencyInterval != nil && *req.FrequencyInterval > 0 {
		interval = *req.FrequencyInterval
	}
	total := req.TotalSessions
	if total < 1 {
		total = 1
	}
	estimated := start.AddDate(0, 0, interval*(total-1))
	p := &treatmentplan.TreatmentPlan{
		ID:                uuid.New(),
		PatientID:         req.PatientID,
		ProfessionalID:    req.ProfessionalID,
		ServiceTypeID:     req.ServiceTypeID,
		Title:             req.Title,
		Objective:         req.Objective,
		TotalSessions:     total,
		CompletedSessions: 0,
		FrequencyType:     freqType,
		FrequencyInterval: interval,
		StartDate:         start,
		EstimatedEndDate:  &estimated,
		Status:            treatmentplan.StatusActive,
		Notes:             req.Notes,
		CreatedBy:         &by,
		CreatedAt:         time.Now(),
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, p.ID)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*treatmentplan.TreatmentPlan, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return p, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*treatmentplan.TreatmentPlan, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	if p.IsTerminal() {
		return nil, ErrPlanTerminal
	}
	if req.ProfessionalID != nil {
		p.ProfessionalID = req.ProfessionalID
	}
	if req.ServiceTypeID != nil {
		p.ServiceTypeID = *req.ServiceTypeID
	}
	if req.Title != nil {
		p.Title = *req.Title
	}
	if req.Objective != nil {
		p.Objective = req.Objective
	}
	if req.TotalSessions != nil && *req.TotalSessions >= 1 {
		p.TotalSessions = *req.TotalSessions
	}
	if req.FrequencyType != nil {
		p.FrequencyType = *req.FrequencyType
		// Auto-derive interval when switching to a standard frequency type
		// (unless the caller explicitly provides one).
		if req.FrequencyInterval == nil {
			p.FrequencyInterval = treatmentplan.FrequencyIntervalDays(*req.FrequencyType, 0)
		}
	}
	if req.FrequencyInterval != nil && *req.FrequencyInterval > 0 {
		p.FrequencyInterval = *req.FrequencyInterval
	}
	if req.StartDate != nil {
		if t, err := time.Parse("2006-01-02", *req.StartDate); err == nil {
			p.StartDate = t
		}
	}
	if req.Notes != nil {
		p.Notes = req.Notes
	}
	estimated := p.StartDate.AddDate(0, 0, p.FrequencyInterval*(p.TotalSessions-1))
	p.EstimatedEndDate = &estimated

	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, id)
}

func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*treatmentplan.TreatmentPlan, error) {
	validStatuses := map[string]bool{
		treatmentplan.StatusActive:    true,
		treatmentplan.StatusPaused:    true,
		treatmentplan.StatusCompleted: true,
		treatmentplan.StatusCancelled: true,
	}
	if !validStatuses[status] {
		return nil, ErrInvalidStatus
	}
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	p.Status = status
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return ErrNotFound
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context, patientID, professionalID, status, serviceTypeID string, limit, offset int) ([]*treatmentplan.TreatmentPlan, int64, error) {
	f := treatmentplan.Filter{Status: status}
	if patientID != "" {
		if id, err := uuid.Parse(patientID); err == nil {
			f.PatientID = &id
		}
	}
	if professionalID != "" {
		if id, err := uuid.Parse(professionalID); err == nil {
			f.ProfessionalID = &id
		}
	}
	if serviceTypeID != "" {
		if id, err := uuid.Parse(serviceTypeID); err == nil {
			f.ServiceTypeID = &id
		}
	}
	return s.repo.List(ctx, f, limit, offset)
}

// GetHistory returns appointments linked to a plan, ordered chronologically.
func (s *Service) GetHistory(ctx context.Context, planID uuid.UUID, limit, offset int) ([]*appointment.Appointment, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	f := appointment.Filter{TreatmentPlanID: &planID}
	return s.apptRepo.List(ctx, f, limit, offset)
}

func (s *Service) GetAlerts(ctx context.Context, professionalID string) ([]*treatmentplan.Alert, error) {
	var pid *uuid.UUID
	if professionalID != "" {
		if id, err := uuid.Parse(professionalID); err == nil {
			pid = &id
		}
	}
	return s.repo.GetAlerts(ctx, pid)
}

// ── Session scheduling ────────────────────────────────────────────────────────

// resolveWorkerID returns the effective worker for session scheduling.
// Plan's ProfessionalID always wins over the request's WorkerID.
func resolveWorkerID(plan *treatmentplan.TreatmentPlan, reqWorkerID *uuid.UUID) *uuid.UUID {
	if plan.ProfessionalID != nil {
		return plan.ProfessionalID
	}
	return reqWorkerID
}

// computeSlotTimes builds the list of scheduled_at times for N sessions starting
// from startAt, spaced by plan.FrequencyInterval days.
func computeSlotTimes(startAt time.Time, n, intervalDays int) []time.Time {
	slots := make([]time.Time, n)
	for i := 0; i < n; i++ {
		slots[i] = startAt.AddDate(0, 0, i*intervalDays)
	}
	return slots
}

// checkSlotAvailability validates a single slot against worker rules + existing bookings.
// Returns (available bool, conflictReason string).
func (s *Service) checkSlotAvailability(ctx context.Context, workerID uuid.UUID, scheduledAt time.Time, duration int) (bool, string) {
	if s.workerRepo != nil {
		rules, err := s.workerRepo.ListAvailabilityRules(ctx, workerID)
		if err == nil && !isWithinAvailability(rules, scheduledAt, duration) {
			return false, fmt.Sprintf("fuera de disponibilidad (%s %s)",
				weekdayName(scheduledAt.Weekday()), scheduledAt.Format("15:04"))
		}
	}
	if conflict, err := s.apptRepo.HasWorkerConflict(ctx, workerID, scheduledAt, duration, nil); err == nil && conflict {
		return false, fmt.Sprintf("conflicto de agenda el %s a las %s",
			scheduledAt.Format("02/01/2006"), scheduledAt.Format("15:04"))
	}
	return true, ""
}

// PreviewSessions computes proposed session dates WITHOUT creating anything.
func (s *Service) PreviewSessions(ctx context.Context, planID uuid.UUID, req PreviewRequest) (*PreviewResult, error) {
	plan, err := s.repo.FindByID(ctx, planID)
	if err != nil {
		return nil, ErrNotFound
	}
	if plan.IsTerminal() {
		return nil, ErrPlanTerminal
	}

	// How many sessions still need to be created?
	active, err := s.apptRepo.CountActiveSessions(ctx, planID)
	if err != nil {
		return nil, err
	}
	toCreate := plan.TotalSessions - plan.CompletedSessions - active
	if toCreate <= 0 {
		return &PreviewResult{TotalToCreate: 0, AllAvailable: true, Slots: []SessionSlotPreview{}}, nil
	}

	startAt, err := parseScheduledAt(req.StartAt)
	if err != nil {
		return nil, errors.New("invalid start_at, expected RFC3339 or YYYY-MM-DDTHH:MM")
	}

	workerID := resolveWorkerID(plan, req.WorkerID)
	duration := coalesceDuration(req.DurationMinutes)
	times := computeSlotTimes(startAt, toCreate, plan.FrequencyInterval)

	slots := make([]SessionSlotPreview, 0, toCreate)
	allAvailable := true
	for i, t := range times {
		sessionNum := plan.CompletedSessions + active + i + 1
		available, reason := true, ""
		if workerID != nil {
			available, reason = s.checkSlotAvailability(ctx, *workerID, t, duration)
		}
		if !available {
			allAvailable = false
		}
		slots = append(slots, SessionSlotPreview{
			SessionNumber:  sessionNum,
			ScheduledAt:    t.Format(time.RFC3339),
			Available:      available,
			ConflictReason: reason,
		})
	}

	return &PreviewResult{
		TotalToCreate: toCreate,
		AllAvailable:  allAvailable,
		Slots:         slots,
	}, nil
}

// GenerateSessions creates appointments for pending sessions of the plan.
// Accepts either auto mode (StartAt) or confirmed mode (Slots with explicit dates).
func (s *Service) GenerateSessions(ctx context.Context, planID uuid.UUID, req GenerateSessionsRequest, by uuid.UUID) ([]*appointment.Appointment, error) {
	plan, err := s.repo.FindByID(ctx, planID)
	if err != nil {
		return nil, ErrNotFound
	}
	if plan.IsTerminal() {
		return nil, ErrPlanTerminal
	}

	// Idempotency: count sessions already scheduled (non-cancelled).
	active, err := s.apptRepo.CountActiveSessions(ctx, planID)
	if err != nil {
		return nil, err
	}
	toCreate := plan.TotalSessions - plan.CompletedSessions - active
	if toCreate <= 0 {
		return []*appointment.Appointment{}, ErrNothingToGenerate
	}

	workerID := resolveWorkerID(plan, req.WorkerID)

	// Resolve slot times: explicit slots override auto-computed ones.
	var slotTimes []time.Time
	if len(req.Slots) > 0 {
		slotTimes = make([]time.Time, 0, len(req.Slots))
		for _, s := range req.Slots {
			t, err := parseScheduledAt(s.ScheduledAt)
			if err != nil {
				return nil, fmt.Errorf("sesión %d: fecha inválida (%s)", s.SessionNumber, s.ScheduledAt)
			}
			slotTimes = append(slotTimes, t)
		}
		// Use only as many as toCreate allows.
		if len(slotTimes) > toCreate {
			slotTimes = slotTimes[:toCreate]
		}
	} else {
		startAt, err := parseScheduledAt(req.StartAt)
		if err != nil {
			return nil, errors.New("invalid start_at, expected RFC3339 or YYYY-MM-DDTHH:MM")
		}
		slotTimes = computeSlotTimes(startAt, toCreate, plan.FrequencyInterval)
	}

	// Assign a single recurring_group_id so all sessions in this batch are linkable.
	groupID := uuid.New()

	// Build appointment batch.
	status := appointment.StatusRequested
	batch := make([]*appointment.Appointment, 0, len(slotTimes))
	for i, t := range slotTimes {
		sessionNum := plan.CompletedSessions + active + i + 1
		n := sessionNum
		batch = append(batch, &appointment.Appointment{
			ID:               uuid.New(),
			PatientID:        plan.PatientID,
			WorkerID:         workerID,
			ServiceTypeID:    plan.ServiceTypeID,
			RecurringGroupID: &groupID,
			ScheduledAt:      t,
			DurationMinutes:  req.DurationMinutes,
			Status:           status,
			TreatmentPlanID:  &plan.ID,
			SessionNumber:    &n,
			CountsAsSession:  true,
			CreatedBy:        &by,
		})
	}

	if err := s.apptRepo.CreateBatch(ctx, batch); err != nil {
		return nil, err
	}

	// Update plan's estimated end date to the last generated session.
	if len(slotTimes) > 0 {
		lastDate := slotTimes[len(slotTimes)-1]
		plan.EstimatedEndDate = &lastDate
		_ = s.repo.Update(ctx, plan)
	}

	// Return enriched appointments.
	result := make([]*appointment.Appointment, 0, len(batch))
	for _, a := range batch {
		if enriched, err := s.apptRepo.FindByID(ctx, a.ID); err == nil {
			result = append(result, enriched)
		}
	}
	return result, nil
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// parseScheduledAt parses a date string using Santiago timezone (same as appointment service).
func parseScheduledAt(s string) (time.Time, error) {
	loc := timeutil.Santiago()
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.In(loc), nil
	}
	for _, f := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(f, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid date format")
}

func coalesceDuration(d *int) int {
	if d != nil && *d > 0 {
		return *d
	}
	return 60
}

// isWithinAvailability checks that scheduledAt + duration fits within one of the worker's active rules.
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
		ruleStart := time.Date(scheduledAt.Year(), scheduledAt.Month(), scheduledAt.Day(),
			ruleStartTime.Hour(), ruleStartTime.Minute(), 0, 0, scheduledAt.Location())
		ruleEnd := time.Date(scheduledAt.Year(), scheduledAt.Month(), scheduledAt.Day(),
			ruleEndTime.Hour(), ruleEndTime.Minute(), 0, 0, scheduledAt.Location())
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

var weekdayNames = []string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}

func weekdayName(d time.Weekday) string {
	return weekdayNames[int(d)%7]
}
