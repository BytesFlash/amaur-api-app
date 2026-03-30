package worker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	appuser "amaur/api/internal/application/user"
	"amaur/api/internal/domain/appointment"
	"amaur/api/internal/domain/program"
	"amaur/api/internal/domain/worker"
	"amaur/api/internal/infrastructure/postgres"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("worker not found")
var ErrUserRequired = errors.New("user_id is required for worker profile")
var ErrUserMustBeProfessional = errors.New("user must have professional role")
var ErrLoginEmailRequired = errors.New("login_email is required when user_id is not provided")
var ErrLoginPasswordRequired = errors.New("login_password is required when user_id is not provided")
var ErrUserAlreadyLinked = errors.New("user already linked to another worker")
var ErrWorkerMustBeAdult = errors.New("el profesional debe ser mayor de edad")
var ErrEmailUsedByAnotherUser = errors.New("el email ya está en uso por otro usuario del sistema")

type CreateWorkerRequest struct {
	UserID            *uuid.UUID `json:"user_id"`
	LoginEmail        *string    `json:"login_email"`
	LoginPassword     *string    `json:"login_password"`
	RUT               *string    `json:"rut"`
	FirstName         string     `json:"first_name"  validate:"required"`
	LastName          string     `json:"last_name"   validate:"required"`
	Email             *string    `json:"email"`
	Phone             *string    `json:"phone"`
	RoleTitle         *string    `json:"role_title"`
	Specialty         *string    `json:"specialty"`
	SpecialtyCodes    []string   `json:"specialty_codes"`
	HireDate          *string    `json:"hire_date"`
	BirthDate         *string    `json:"birth_date"`
	AvailabilityNotes *string    `json:"availability_notes"`
	InternalNotes     *string    `json:"internal_notes"`
}

type UpdateWorkerRequest struct {
	RUT               *string  `json:"rut"`
	FirstName         *string  `json:"first_name"`
	LastName          *string  `json:"last_name"`
	Email             *string  `json:"email"`
	Phone             *string  `json:"phone"`
	RoleTitle         *string  `json:"role_title"`
	Specialty         *string  `json:"specialty"`
	SpecialtyCodes    []string `json:"specialty_codes"`
	HireDate          *string  `json:"hire_date"`
	BirthDate         *string  `json:"birth_date"`
	TerminationDate   *string  `json:"termination_date"`
	IsActive          *bool    `json:"is_active"`
	AvailabilityNotes *string  `json:"availability_notes"`
	InternalNotes     *string  `json:"internal_notes"`
}

type Service struct {
	repo        worker.Repository
	userRepo    *postgres.UserRepository
	userSvc     *appuser.Service
	apptRepo    appointment.Repository
	programRepo program.Repository
}

func NewService(repo worker.Repository, userRepo *postgres.UserRepository, userSvc *appuser.Service) *Service {
	return &Service{repo: repo, userRepo: userRepo, userSvc: userSvc}
}

// WithAppointmentRepo attaches an appointment repo so GetWorkerSlots can check occupancy.
func (s *Service) WithAppointmentRepo(r appointment.Repository) *Service {
	s.apptRepo = r
	return s
}

// WithProgramRepo attaches the program repo so group agenda sessions are counted in availability.
func (s *Service) WithProgramRepo(r program.Repository) *Service {
	s.programRepo = r
	return s
}

func (s *Service) Create(ctx context.Context, req CreateWorkerRequest, createdBy uuid.UUID) (*worker.Worker, error) {
	// Adult validation (must be 18+)
	if req.BirthDate != nil {
		bd := parseDatePtr(req.BirthDate)
		if bd != nil && !isAdult(*bd) {
			return nil, ErrWorkerMustBeAdult
		}
	}

	resolvedUserID := req.UserID
	if resolvedUserID == nil {
		loginEmail := ""
		if req.LoginEmail != nil {
			loginEmail = strings.TrimSpace(*req.LoginEmail)
		}
		if loginEmail == "" && req.Email != nil {
			loginEmail = strings.TrimSpace(*req.Email)
		}
		if loginEmail == "" {
			return nil, ErrLoginEmailRequired
		}
		if req.LoginPassword == nil || strings.TrimSpace(*req.LoginPassword) == "" {
			return nil, ErrLoginPasswordRequired
		}

		roleID, err := s.userSvc.GetRoleIDByName(ctx, "professional")
		if err != nil {
			return nil, err
		}
		createdUser, err := s.userSvc.Create(ctx, appuser.CreateUserRequest{
			Email:     loginEmail,
			Password:  *req.LoginPassword,
			FirstName: req.FirstName,
			LastName:  req.LastName,
			RoleIDs:   []uuid.UUID{roleID},
		}, createdBy)
		if err != nil {
			return nil, err
		}
		resolvedUserID = &createdUser.ID
	}
	if _, err := s.repo.FindByUserID(ctx, *resolvedUserID); err == nil {
		return nil, ErrUserAlreadyLinked
	}

	roles, err := s.userRepo.GetRoleNames(ctx, *resolvedUserID)
	if err != nil {
		return nil, err
	}
	isProfessional := false
	for _, rn := range roles {
		if rn == "professional" {
			isProfessional = true
			break
		}
	}
	if !isProfessional {
		return nil, ErrUserMustBeProfessional
	}

	// Cross-person email check: if a contact email is provided, ensure it isn't
	// the login email of a different user.
	if req.Email != nil && *req.Email != "" {
		existingUser, _ := s.userRepo.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(*req.Email)))
		if existingUser != nil && (resolvedUserID == nil || existingUser.ID != *resolvedUserID) {
			return nil, ErrEmailUsedByAnotherUser
		}
	}

	w := &worker.Worker{
		ID:                uuid.New(),
		UserID:            resolvedUserID,
		RUT:               req.RUT,
		FirstName:         req.FirstName,
		LastName:          req.LastName,
		Email:             req.Email,
		Phone:             req.Phone,
		RoleTitle:         req.RoleTitle,
		Specialty:         req.Specialty,
		HireDate:          parseDatePtr(req.HireDate),
		BirthDate:         parseDatePtr(req.BirthDate),
		IsActive:          true,
		AvailabilityNotes: req.AvailabilityNotes,
		InternalNotes:     req.InternalNotes,
		CreatedBy:         &createdBy,
	}
	if err := s.repo.Create(ctx, w); err != nil {
		if resolvedUserID != nil && req.UserID == nil {
			_ = s.userSvc.Delete(ctx, *resolvedUserID)
		}
		return nil, err
	}

	// Set specialties from catalog if provided.
	if len(req.SpecialtyCodes) > 0 {
		_ = s.repo.SetWorkerSpecialties(ctx, w.ID, req.SpecialtyCodes, createdBy)
	}
	w.Specialties, _ = s.repo.GetWorkerSpecialties(ctx, w.ID)
	return w, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*worker.Worker, error) {
	w, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	w.Specialties, _ = s.repo.GetWorkerSpecialties(ctx, w.ID)
	return w, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateWorkerRequest, updatedBy uuid.UUID) (*worker.Worker, error) {
	w, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	now := time.Now()
	if req.FirstName != nil {
		w.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		w.LastName = *req.LastName
	}
	if req.RUT != nil {
		w.RUT = req.RUT
	}
	if req.Email != nil {
		w.Email = req.Email
	}
	if req.Phone != nil {
		w.Phone = req.Phone
	}
	if req.RoleTitle != nil {
		w.RoleTitle = req.RoleTitle
	}
	if req.Specialty != nil {
		w.Specialty = req.Specialty
	}
	if req.HireDate != nil {
		w.HireDate = parseDatePtr(req.HireDate)
	}
	if req.BirthDate != nil {
		bd := parseDatePtr(req.BirthDate)
		if bd != nil && !isAdult(*bd) {
			return nil, ErrWorkerMustBeAdult
		}
		w.BirthDate = bd
	}
	if req.Email != nil && *req.Email != "" {
		existingUser, _ := s.userRepo.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(*req.Email)))
		if existingUser != nil && (w.UserID == nil || existingUser.ID != *w.UserID) {
			return nil, ErrEmailUsedByAnotherUser
		}
	}
	if req.TerminationDate != nil {
		w.TerminationDate = parseDatePtr(req.TerminationDate)
	}
	if req.IsActive != nil {
		w.IsActive = *req.IsActive
	}
	if req.AvailabilityNotes != nil {
		w.AvailabilityNotes = req.AvailabilityNotes
	}
	if req.InternalNotes != nil {
		w.InternalNotes = req.InternalNotes
	}
	w.UpdatedAt = &now
	w.UpdatedBy = &updatedBy
	if err := s.repo.Update(ctx, w); err != nil {
		return nil, err
	}

	// Replace specialties if the field was provided (nil slice = no change).
	if req.SpecialtyCodes != nil {
		_ = s.repo.SetWorkerSpecialties(ctx, w.ID, req.SpecialtyCodes, updatedBy)
	}
	w.Specialties, _ = s.repo.GetWorkerSpecialties(ctx, w.ID)
	return w, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return ErrNotFound
	}
	return s.repo.SoftDelete(ctx, id)
}

func (s *Service) List(ctx context.Context, search string, specialtyCode string, onlyActive bool, limit, offset int) ([]*worker.Worker, int64, error) {
	// Specialties are bulk-loaded inside the repo List call.
	return s.repo.List(ctx, search, specialtyCode, onlyActive, limit, offset)
}

// ListSpecialties returns the active specialty catalog.
func (s *Service) ListSpecialties(ctx context.Context) ([]worker.SpecialtyItem, error) {
	return s.repo.ListSpecialties(ctx)
}

// CreateSpecialty adds an entry to the specialty catalog.
func (s *Service) CreateSpecialty(ctx context.Context, code, name string) (*worker.SpecialtyItem, error) {
	item := worker.SpecialtyItem{Code: code, Name: name}
	if err := s.repo.CreateSpecialty(ctx, item); err != nil {
		return nil, err
	}
	return &item, nil
}

// DeleteSpecialty removes a specialty from the catalog (fails if referenced by workers or service types).
func (s *Service) DeleteSpecialty(ctx context.Context, code string) error {
	return s.repo.DeleteSpecialty(ctx, code)
}

// SetWorkerSpecialties replaces all specialties for the given worker.
func (s *Service) SetWorkerSpecialties(ctx context.Context, workerID uuid.UUID, codes []string, setBy uuid.UUID) ([]worker.SpecialtyItem, error) {
	if _, err := s.repo.FindByID(ctx, workerID); err != nil {
		return nil, ErrNotFound
	}
	if err := s.repo.SetWorkerSpecialties(ctx, workerID, codes, setBy); err != nil {
		return nil, err
	}
	return s.repo.GetWorkerSpecialties(ctx, workerID)
}

// --- availability rules ---

type AvailabilityRuleInput struct {
	Weekday   int16  `json:"weekday"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// GetAvailabilityRules returns current rules for a worker.
func (s *Service) GetAvailabilityRules(ctx context.Context, workerID uuid.UUID) ([]*worker.AvailabilityRule, error) {
	if _, err := s.repo.FindByID(ctx, workerID); err != nil {
		return nil, ErrNotFound
	}
	return s.repo.ListAvailabilityRules(ctx, workerID)
}

// GetWorkerCalendar returns per-day availability and booking summaries for a month.
// monthStr must be "YYYY-MM"; if empty defaults to current month.
func (s *Service) GetWorkerCalendar(ctx context.Context, workerID uuid.UUID, monthStr string) ([]*worker.DayCalendar, error) {
	if _, err := s.repo.FindByID(ctx, workerID); err != nil {
		return nil, ErrNotFound
	}
	if monthStr == "" {
		monthStr = time.Now().Format("2006-01")
	}
	firstDay, err := time.Parse("2006-01", monthStr)
	if err != nil {
		return nil, fmt.Errorf("month must be YYYY-MM, got %s", monthStr)
	}
	// Last day is first day of next month minus 1.
	firstNextMonth := firstDay.AddDate(0, 1, 0)
	lastDay := firstNextMonth.AddDate(0, 0, -1)

	rules, err := s.repo.ListAvailabilityRules(ctx, workerID)
	if err != nil {
		return nil, err
	}

	// Fetch appointments for the month.
	apptsByDate := make(map[string][]*appointment.Appointment)
	if s.apptRepo != nil {
		appts, _, err := s.apptRepo.List(ctx, appointment.Filter{
			WorkerID: &workerID,
			DateFrom: &firstDay,
			DateTo:   &firstNextMonth,
		}, 1000, 0)
		if err == nil {
			for _, a := range appts {
				if a.Status == "confirmed" || a.Status == "scheduled" || a.Status == "completed" {
					dateKey := a.ScheduledAt.Format("2006-01-02")
					apptsByDate[dateKey] = append(apptsByDate[dateKey], a)
				}
			}
		}
	}

	// Fetch group agenda services for the month.
	type groupBlock struct {
		startTime string
		duration  int
		label     string
	}
	groupByDate := make(map[string][]groupBlock)
	if s.programRepo != nil {
		groupSvcs, err := s.programRepo.ListAgendaServicesByWorker(ctx, workerID, firstDay, firstNextMonth)
		if err == nil {
			for _, svc := range groupSvcs {
				dateKey := svc.ScheduledDate.Format("2006-01-02")
				dur := 60
				if svc.PlannedDurationMinutes != nil {
					dur = *svc.PlannedDurationMinutes
				}
				startT := ""
				if svc.PlannedStartTime != nil {
					startT = *svc.PlannedStartTime
				}
				lbl := "Servicio grupal"
				if svc.ServiceTypeName != nil {
					lbl = *svc.ServiceTypeName
				}
				groupByDate[dateKey] = append(groupByDate[dateKey], groupBlock{startTime: startT, duration: dur, label: lbl})
			}
		}
	}

	result := make([]*worker.DayCalendar, 0, lastDay.Day())
	for d := firstDay; !d.After(lastDay); d = d.AddDate(0, 0, 1) {
		wd := int16(d.Weekday())
		dateStr := d.Format("2006-01-02")

		// Sum available minutes from matching rules.
		totalMinutes := 0
		for _, rule := range rules {
			if rule.Weekday != wd || !rule.IsActive {
				continue
			}
			rStart, err1 := time.Parse("15:04", rule.StartTime)
			rEnd, err2 := time.Parse("15:04", rule.EndTime)
			if err1 != nil || err2 != nil {
				continue
			}
			totalMinutes += int(rEnd.Sub(rStart).Minutes())
		}

		// Aggregate appointments.
		bookedMinutes := 0
		var summs []worker.ApptSummary
		for _, a := range apptsByDate[dateStr] {
			dur := 60
			if a.DurationMinutes != nil {
				dur = *a.DurationMinutes
			}
			bookedMinutes += dur
			label := ""
			if a.PatientName != nil {
				label = *a.PatientName
			}
			if label == "" && a.ServiceTypeName != nil {
				label = *a.ServiceTypeName
			}
			summs = append(summs, worker.ApptSummary{
				ScheduledAt:     a.ScheduledAt.Format("15:04"),
				DurationMinutes: dur,
				Type:            "individual",
				Label:           label,
			})
		}
		// Add group agenda sessions to booked time.
		for _, g := range groupByDate[dateStr] {
			bookedMinutes += g.duration
			summs = append(summs, worker.ApptSummary{
				ScheduledAt:     g.startTime,
				DurationMinutes: g.duration,
				Type:            "group",
				Label:           g.label,
			})
		}

		availMinutes := totalMinutes - bookedMinutes
		if availMinutes < 0 {
			availMinutes = 0
		}
		result = append(result, &worker.DayCalendar{
			Date:             dateStr,
			TotalMinutes:     totalMinutes,
			AvailableMinutes: availMinutes,
			BookedMinutes:    bookedMinutes,
			Appointments:     summs,
		})
	}
	return result, nil
}

// SetAvailabilityRules replaces rules for a worker.
func (s *Service) SetAvailabilityRules(ctx context.Context, workerID uuid.UUID, inputs []AvailabilityRuleInput, setBy uuid.UUID) ([]*worker.AvailabilityRule, error) {
	if _, err := s.repo.FindByID(ctx, workerID); err != nil {
		return nil, ErrNotFound
	}
	rules := make([]*worker.AvailabilityRule, 0, len(inputs))
	for _, in := range inputs {
		rules = append(rules, &worker.AvailabilityRule{
			ID:        uuid.New(),
			WorkerID:  workerID,
			Weekday:   in.Weekday,
			StartTime: in.StartTime,
			EndTime:   in.EndTime,
			IsActive:  true,
			CreatedBy: &setBy,
		})
	}
	if err := s.repo.ReplaceAvailabilityRules(ctx, workerID, rules); err != nil {
		return nil, err
	}
	return s.repo.ListAvailabilityRules(ctx, workerID)
}

// GetWorkerSlots calculates available time blocks for a worker for a given week.
// weekStartStr must be "YYYY-MM-DD" (any day works, we normalise to Monday).
// duration specifies slot length in minutes.
func (s *Service) GetWorkerSlots(ctx context.Context, workerID uuid.UUID, weekStartStr string, duration int) ([]worker.TimeSlot, error) {
	if _, err := s.repo.FindByID(ctx, workerID); err != nil {
		return nil, ErrNotFound
	}
	weekStart, err := time.Parse("2006-01-02", weekStartStr)
	if err != nil {
		return nil, fmt.Errorf("week_start must be YYYY-MM-DD")
	}
	// Normalise to Monday (ISO week).
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	weekEnd := weekStart.AddDate(0, 0, 7)

	rules, err := s.repo.ListAvailabilityRules(ctx, workerID)
	if err != nil {
		return nil, err
	}

	// Fetch existing appointments that overlap this week.
	var occupied []*appointment.Appointment
	if s.apptRepo != nil {
		from := weekStart
		to := weekEnd
		appts, _, err := s.apptRepo.List(ctx, appointment.Filter{
			WorkerID: &workerID,
			DateFrom: &from,
			DateTo:   &to,
		}, 500, 0)
		if err == nil {
			for _, a := range appts {
				if a.Status == "confirmed" || a.Status == "scheduled" {
					occupied = append(occupied, a)
				}
			}
		}
	}

	// Build occupied intervals from group agenda services, expressed as absolute UTC times.
	type absInterval struct{ start, end time.Time }
	var groupOccupied []absInterval
	if s.programRepo != nil {
		groupSvcs, err := s.programRepo.ListAgendaServicesByWorker(ctx, workerID, weekStart, weekEnd)
		if err == nil {
			for _, svc := range groupSvcs {
				if svc.PlannedStartTime == nil || svc.PlannedDurationMinutes == nil {
					continue
				}
				t, err := time.Parse("15:04", *svc.PlannedStartTime)
				if err != nil {
					continue
				}
				d := svc.ScheduledDate
				start := time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), 0, 0, time.UTC)
				end := start.Add(time.Duration(*svc.PlannedDurationMinutes) * time.Minute)
				groupOccupied = append(groupOccupied, absInterval{start, end})
			}
		}
	}

	var slots []worker.TimeSlot
	for day := 0; day < 7; day++ {
		date := weekStart.AddDate(0, 0, day)
		wd := int16(date.Weekday()) // 0=Sunday

		for _, rule := range rules {
			if rule.Weekday != wd {
				continue
			}
			// Parse rule times.
			ruleStart, err := time.Parse("15:04", rule.StartTime)
			if err != nil {
				continue
			}
			ruleEnd, err := time.Parse("15:04", rule.EndTime)
			if err != nil {
				continue
			}

			cur := ruleStart
			for cur.Add(time.Duration(duration)*time.Minute).Before(ruleEnd) ||
				cur.Add(time.Duration(duration)*time.Minute).Equal(ruleEnd) {
				slotEnd := cur.Add(time.Duration(duration) * time.Minute)
				// Build absolute times for occupancy check.
				slotStartAbs := time.Date(date.Year(), date.Month(), date.Day(),
					cur.Hour(), cur.Minute(), 0, 0, time.UTC)
				slotEndAbs := slotStartAbs.Add(time.Duration(duration) * time.Minute)

				available := true
				for _, a := range occupied {
					apptEnd := a.ScheduledAt.Add(time.Duration(coalesce(a.DurationMinutes, &duration)) * time.Minute)
					if a.ScheduledAt.Before(slotEndAbs) && apptEnd.After(slotStartAbs) {
						available = false
						break
					}
				}
				if available {
					for _, g := range groupOccupied {
						if g.start.Before(slotEndAbs) && g.end.After(slotStartAbs) {
							available = false
							break
						}
					}
				}

				slots = append(slots, worker.TimeSlot{
					Date:      date.Format("2006-01-02"),
					Weekday:   int(wd),
					StartTime: cur.Format("15:04"),
					EndTime:   slotEnd.Format("15:04"),
					Available: available,
				})
				cur = slotEnd
			}
		}
	}
	return slots, nil
}

func coalesce(p *int, fallback *int) int {
	if p != nil {
		return *p
	}
	if fallback != nil {
		return *fallback
	}
	return 60
}

func isAdult(birthDate time.Time) bool {
	now := time.Now()
	age := now.Year() - birthDate.Year()
	if now.Month() < birthDate.Month() || (now.Month() == birthDate.Month() && now.Day() < birthDate.Day()) {
		age--
	}
	return age >= 18
}

func parseDatePtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, *s)
		if err != nil {
			return nil
		}
	}
	return &t
}
