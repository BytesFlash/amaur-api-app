package program

import (
	"context"
	"errors"
	"time"

	"amaur/api/internal/domain/caresession"
	"amaur/api/internal/domain/contract"
	"amaur/api/internal/domain/program"

	"github.com/google/uuid"
)

var (
	ErrProgramNotFound             = errors.New("program not found")
	ErrContractNotFound            = errors.New("contract not found")
	ErrContractCompanyMismatch     = errors.New("contract does not belong to company")
	ErrInvalidDateRange            = errors.New("invalid date range")
	ErrOutsideContractRange        = errors.New("program date range is outside contract range")
	ErrAgendaServiceNotFound       = errors.New("agenda service not found")
	ErrParticipantsOutsideCompany  = errors.New("one or more participants are not associated to agenda company")
	ErrAgendaServiceWorkerRequired = errors.New("agenda service must have a worker assigned before completion")
)

type ScheduleRuleInput struct {
	Weekday                int16      `json:"weekday"`
	StartTime              string     `json:"start_time"`
	DurationMinutes        int        `json:"duration_minutes"`
	FrequencyIntervalWeeks int        `json:"frequency_interval_weeks"`
	MaxOccurrences         *int       `json:"max_occurrences"`
	ServiceTypeID          *uuid.UUID `json:"service_type_id"`
	WorkerID               *uuid.UUID `json:"worker_id"`
}

type CreateProgramRequest struct {
	CompanyID  uuid.UUID              `json:"company_id"`
	ContractID uuid.UUID              `json:"contract_id"`
	Name       string                 `json:"name"`
	StartDate  string                 `json:"start_date"`
	EndDate    *string                `json:"end_date"`
	Status     *program.ProgramStatus `json:"status"`
	Notes      *string                `json:"notes"`
	Rules      []ScheduleRuleInput    `json:"rules"`
}

type UpdateProgramRequest struct {
	Name      *string                `json:"name"`
	StartDate *string                `json:"start_date"`
	EndDate   *string                `json:"end_date"`
	Status    *program.ProgramStatus `json:"status"`
	Notes     *string                `json:"notes"`
	Rules     *[]ScheduleRuleInput   `json:"rules"`
}

type CreateAgendaServiceRequest struct {
	AgendaID               uuid.UUID  `json:"agenda_id"`
	ServiceTypeID          uuid.UUID  `json:"service_type_id"`
	WorkerID               *uuid.UUID `json:"worker_id"`
	PlannedStartTime       *string    `json:"planned_start_time"`
	PlannedDurationMinutes *int       `json:"planned_duration_minutes"`
	Notes                  *string    `json:"notes"`
}

type ParticipantInput struct {
	PatientID  uuid.UUID `json:"patient_id"`
	Attended   bool      `json:"attended"`
	AttendedAt *string   `json:"attended_at"`
	Notes      *string   `json:"notes"`
}

type Service struct {
	repo         program.Repository
	contractRepo contract.Repository
	careRepo     caresession.Repository
}

func NewService(repo program.Repository, contractRepo contract.Repository, careRepo caresession.Repository) *Service {
	return &Service{repo: repo, contractRepo: contractRepo, careRepo: careRepo}
}

func (s *Service) CreateProgram(ctx context.Context, req CreateProgramRequest, createdBy uuid.UUID) (*program.CompanyProgram, error) {
	ct, err := s.contractRepo.FindByID(ctx, req.ContractID)
	if err != nil {
		return nil, ErrContractNotFound
	}
	if ct.CompanyID != req.CompanyID {
		return nil, ErrContractCompanyMismatch
	}

	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return nil, ErrInvalidDateRange
	}
	endDate := parseDatePtr(req.EndDate)
	if endDate != nil && startDate.After(*endDate) {
		return nil, ErrInvalidDateRange
	}
	if !dateRangeWithinContract(startDate, endDate, ct.StartDate, ct.EndDate) {
		return nil, ErrOutsideContractRange
	}

	status := program.StatusDraft
	if req.Status != nil {
		status = *req.Status
	}

	p := &program.CompanyProgram{
		ID:         uuid.New(),
		CompanyID:  req.CompanyID,
		ContractID: req.ContractID,
		Name:       req.Name,
		StartDate:  startDate,
		EndDate:    endDate,
		Status:     status,
		Notes:      req.Notes,
		CreatedBy:  &createdBy,
	}
	if err := s.repo.CreateProgram(ctx, p); err != nil {
		return nil, err
	}

	if len(req.Rules) > 0 {
		rules := make([]*program.ScheduleRule, 0, len(req.Rules))
		for _, r := range req.Rules {
			rule := &program.ScheduleRule{
				ID:                     uuid.New(),
				ProgramID:              p.ID,
				Weekday:                r.Weekday,
				StartTime:              r.StartTime,
				DurationMinutes:        r.DurationMinutes,
				FrequencyIntervalWeeks: maxInt(r.FrequencyIntervalWeeks, 1),
				MaxOccurrences:         r.MaxOccurrences,
				ServiceTypeID:          r.ServiceTypeID,
				WorkerID:               r.WorkerID,
				CreatedBy:              &createdBy,
			}
			rules = append(rules, rule)
		}
		if err := s.repo.CreateScheduleRules(ctx, rules); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (s *Service) GetProgramByID(ctx context.Context, id uuid.UUID) (*program.CompanyProgram, []*program.ScheduleRule, error) {
	p, err := s.repo.GetProgramByID(ctx, id)
	if err != nil {
		return nil, nil, ErrProgramNotFound
	}
	rules, err := s.repo.ListScheduleRules(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return p, rules, nil
}

func (s *Service) UpdateProgram(ctx context.Context, id uuid.UUID, req UpdateProgramRequest, updatedBy uuid.UUID) (*program.CompanyProgram, error) {
	p, err := s.repo.GetProgramByID(ctx, id)
	if err != nil {
		return nil, ErrProgramNotFound
	}

	ct, err := s.contractRepo.FindByID(ctx, p.ContractID)
	if err != nil {
		return nil, ErrContractNotFound
	}

	startDate := p.StartDate
	if req.StartDate != nil {
		parsedStartDate, parseErr := parseDate(*req.StartDate)
		if parseErr != nil {
			return nil, ErrInvalidDateRange
		}
		startDate = parsedStartDate
	}
	endDate := p.EndDate
	if req.EndDate != nil {
		endDate = parseDatePtr(req.EndDate)
	}
	if endDate != nil && startDate.After(*endDate) {
		return nil, ErrInvalidDateRange
	}
	if !dateRangeWithinContract(startDate, endDate, ct.StartDate, ct.EndDate) {
		return nil, ErrOutsideContractRange
	}

	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Status != nil {
		p.Status = *req.Status
	}
	if req.Notes != nil {
		p.Notes = req.Notes
	}
	p.StartDate = startDate
	p.EndDate = endDate
	p.UpdatedBy = &updatedBy
	now := time.Now()
	p.UpdatedAt = &now

	if err := s.repo.UpdateProgram(ctx, p); err != nil {
		return nil, err
	}

	if req.Rules != nil {
		rules := make([]*program.ScheduleRule, 0, len(*req.Rules))
		for _, r := range *req.Rules {
			rule := &program.ScheduleRule{
				ID:                     uuid.New(),
				ProgramID:              p.ID,
				Weekday:                r.Weekday,
				StartTime:              r.StartTime,
				DurationMinutes:        r.DurationMinutes,
				FrequencyIntervalWeeks: maxInt(r.FrequencyIntervalWeeks, 1),
				MaxOccurrences:         r.MaxOccurrences,
				ServiceTypeID:          r.ServiceTypeID,
				WorkerID:               r.WorkerID,
				CreatedBy:              &updatedBy,
			}
			rules = append(rules, rule)
		}
		if err := s.repo.ReplaceScheduleRules(ctx, p.ID, rules); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (s *Service) ListPrograms(ctx context.Context, companyIDStr, contractIDStr, status, dateFrom, dateTo string, limit, offset int) ([]*program.CompanyProgram, int64, error) {
	f := program.Filter{}
	if companyIDStr != "" {
		if id, err := uuid.Parse(companyIDStr); err == nil {
			f.CompanyID = &id
		}
	}
	if contractIDStr != "" {
		if id, err := uuid.Parse(contractIDStr); err == nil {
			f.ContractID = &id
		}
	}
	if status != "" {
		f.Status = program.ProgramStatus(status)
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
	return s.repo.ListPrograms(ctx, f, limit, offset)
}

func (s *Service) CreateAgendaService(ctx context.Context, req CreateAgendaServiceRequest) (*program.AgendaService, error) {
	svc := &program.AgendaService{
		ID:                     uuid.New(),
		AgendaID:               req.AgendaID,
		ServiceTypeID:          req.ServiceTypeID,
		WorkerID:               req.WorkerID,
		PlannedStartTime:       req.PlannedStartTime,
		PlannedDurationMinutes: req.PlannedDurationMinutes,
		Status:                 program.AgendaServicePlanned,
		Notes:                  req.Notes,
	}
	if err := s.repo.CreateAgendaServices(ctx, []*program.AgendaService{svc}); err != nil {
		return nil, err
	}
	return svc, nil
}

func (s *Service) UpsertAgendaServiceParticipants(ctx context.Context, agendaServiceID uuid.UUID, participants []ParticipantInput, createdBy uuid.UUID) error {
	patientIDs := make([]uuid.UUID, 0, len(participants))
	for _, p := range participants {
		patientIDs = append(patientIDs, p.PatientID)
	}
	outside, err := s.repo.PatientIDsOutsideAgendaCompany(ctx, agendaServiceID, patientIDs)
	if err != nil {
		return err
	}
	if len(outside) > 0 {
		return ErrParticipantsOutsideCompany
	}

	items := make([]*program.AgendaServiceParticipant, 0, len(participants))
	for _, p := range participants {
		var attendedAt *time.Time
		if p.AttendedAt != nil && *p.AttendedAt != "" {
			t, err := time.Parse(time.RFC3339, *p.AttendedAt)
			if err == nil {
				attendedAt = &t
			}
		}
		item := &program.AgendaServiceParticipant{
			ID:              uuid.New(),
			AgendaServiceID: agendaServiceID,
			PatientID:       p.PatientID,
			Attended:        p.Attended,
			AttendedAt:      attendedAt,
			Notes:           p.Notes,
			CreatedBy:       &createdBy,
		}
		items = append(items, item)
	}
	return s.repo.UpsertParticipants(ctx, items)
}

func (s *Service) CompleteAgendaService(ctx context.Context, agendaServiceID uuid.UUID, completedBy uuid.UUID) error {
	svc, err := s.repo.GetAgendaServiceByID(ctx, agendaServiceID)
	if err != nil {
		return ErrAgendaServiceNotFound
	}
	if svc.WorkerID == nil {
		return ErrAgendaServiceWorkerRequired
	}
	ctxData, err := s.repo.GetAgendaContextByServiceID(ctx, agendaServiceID)
	if err != nil {
		return err
	}
	participants, err := s.repo.ListParticipants(ctx, agendaServiceID)
	if err != nil {
		return err
	}

	attended := make([]*program.AgendaServiceParticipant, 0)
	ids := make([]uuid.UUID, 0)
	for _, p := range participants {
		if p.Attended {
			attended = append(attended, p)
			ids = append(ids, p.PatientID)
		}
	}

	outside, err := s.repo.PatientIDsOutsideAgendaCompany(ctx, agendaServiceID, ids)
	if err != nil {
		return err
	}
	if len(outside) > 0 {
		return ErrParticipantsOutsideCompany
	}

	for _, p := range attended {
		sessionTime := svc.PlannedStartTime
		if sessionTime == nil {
			sessionTime = ctxData.ScheduledStart
		}
		cs := &caresession.CareSession{
			VisitID:          &svc.AgendaID,
			PatientID:        p.PatientID,
			WorkerID:         *svc.WorkerID,
			ServiceTypeID:    svc.ServiceTypeID,
			CompanyID:        &ctxData.CompanyID,
			SessionType:      "company_program",
			SessionDate:      ctxData.ScheduledDate,
			SessionTime:      sessionTime,
			DurationMinutes:  svc.PlannedDurationMinutes,
			Status:           "completed",
			FollowUpRequired: false,
			CreatedBy:        &completedBy,
		}
		if err := s.careRepo.Create(ctx, cs); err != nil {
			return err
		}
	}

	now := time.Now()
	svc.Status = program.AgendaServiceCompleted
	svc.CompletedBy = &completedBy
	svc.CompletedAt = &now
	return s.repo.UpdateAgendaService(ctx, svc)
}

func parseDate(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

func parseDatePtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := parseDate(*s)
	if err != nil {
		return nil
	}
	return &t
}

func dateRangeWithinContract(programStart time.Time, programEnd *time.Time, contractStart time.Time, contractEnd *time.Time) bool {
	if programStart.Before(contractStart) {
		return false
	}
	if contractEnd == nil {
		return true
	}
	if programStart.After(*contractEnd) {
		return false
	}
	if programEnd != nil && programEnd.After(*contractEnd) {
		return false
	}
	return true
}

func maxInt(v int, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}

// GetProgramAgendas returns all agendas linked to a program, enriched with their services.
func (s *Service) GetProgramAgendas(ctx context.Context, programID uuid.UUID) ([]*program.AgendaWithServices, error) {
	if _, err := s.repo.GetProgramByID(ctx, programID); err != nil {
		return nil, ErrProgramNotFound
	}
	return s.repo.ListProgramAgendas(ctx, programID)
}

// GetAgendaServices returns the services for a single agenda (no enrichment needed for simple list).
func (s *Service) GetAgendaServices(ctx context.Context, agendaID uuid.UUID) ([]*program.AgendaService, error) {
	return s.repo.ListAgendaServices(ctx, agendaID)
}

// GetParticipantsDetail returns participants for a service, enriched with patient names.
func (s *Service) GetParticipantsDetail(ctx context.Context, agendaServiceID uuid.UUID) ([]*program.ParticipantDetail, error) {
	return s.repo.ListParticipantsDetail(ctx, agendaServiceID)
}

// GenerateAgendas reads program schedule rules and creates agenda rows + services for each occurrence.
// Returns the count and list of created agenda IDs.
func (s *Service) GenerateAgendas(ctx context.Context, programID uuid.UUID, by uuid.UUID) (int, []uuid.UUID, error) {
	p, rules, err := s.GetProgramByID(ctx, programID)
	if err != nil {
		return 0, nil, ErrProgramNotFound
	}
	if len(rules) == 0 {
		return 0, []uuid.UUID{}, nil
	}

	endDate := p.EndDate
	if endDate == nil {
		// Default to 1 year from start if no end is set
		d := p.StartDate.AddDate(1, 0, 0)
		endDate = &d
	}

	var created []uuid.UUID
	for _, rule := range rules {
		dates := occurrences(p.StartDate, *endDate, int(rule.Weekday), rule.FrequencyIntervalWeeks)
		for _, date := range dates {
			startStr := rule.StartTime
			agendaID, err := s.repo.CreateAgenda(ctx, p.CompanyID, &p.ContractID, date, &startStr, by)
			if err != nil {
				continue // skip failures (e.g. duplicates)
			}
			_ = s.repo.LinkProgramAgenda(ctx, p.ID, agendaID)
			if rule.ServiceTypeID != nil {
				workerID := rule.WorkerID
				svc := &program.AgendaService{
					ID:                     uuid.New(),
					AgendaID:               agendaID,
					ServiceTypeID:          *rule.ServiceTypeID,
					WorkerID:               workerID,
					PlannedStartTime:       &startStr,
					PlannedDurationMinutes: &rule.DurationMinutes,
					Status:                 program.AgendaServicePlanned,
				}
				_ = s.repo.CreateAgendaServices(ctx, []*program.AgendaService{svc})
			}
			created = append(created, agendaID)
		}
	}
	return len(created), created, nil
}

// occurrences returns all dates matching weekday (0=Sun) from start to end, stepping by freqWeeks weeks.
func occurrences(start, end time.Time, weekday, freqWeeks int) []time.Time {
	if freqWeeks < 1 {
		freqWeeks = 1
	}
	// Find first occurrence of weekday on or after start
	cur := start
	for int(cur.Weekday()) != weekday {
		cur = cur.AddDate(0, 0, 1)
	}
	var result []time.Time
	for !cur.After(end) {
		result = append(result, cur)
		cur = cur.AddDate(0, 0, freqWeeks*7)
	}
	return result
}
