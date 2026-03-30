package caresession

import (
	"context"
	"errors"
	"time"

	"amaur/api/internal/domain/caresession"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("care session not found")

// ── DTOs ─────────────────────────────────────────────────────────────────────

type CreateCareSessionRequest struct {
	VisitID           *uuid.UUID `json:"visit_id"`
	PatientID         uuid.UUID  `json:"patient_id"`
	WorkerID          uuid.UUID  `json:"worker_id"`
	ServiceTypeID     uuid.UUID  `json:"service_type_id"`
	CompanyID         *uuid.UUID `json:"company_id"`
	ContractServiceID *uuid.UUID `json:"contract_service_id"`
	SessionType       string     `json:"session_type"` // company_visit | particular
	SessionDate       string     `json:"session_date"`
	SessionTime       *string    `json:"session_time"`
	DurationMinutes   *int       `json:"duration_minutes"`
	Notes             *string    `json:"notes"`
}

type UpdateCareSessionRequest struct {
	Status           *string `json:"status"`
	DurationMinutes  *int    `json:"duration_minutes"`
	ChiefComplaint   *string `json:"chief_complaint"`
	Subjective       *string `json:"subjective"`
	Objective        *string `json:"objective"`
	Assessment       *string `json:"assessment"`
	Plan             *string `json:"plan"`
	Notes            *string `json:"notes"`
	FollowUpRequired *bool   `json:"follow_up_required"`
	FollowUpStatus   *string `json:"follow_up_status"`
	FollowUpDate     *string `json:"follow_up_date"`
	FollowUpNotes    *string `json:"follow_up_notes"`
}

type CreateGroupSessionRequest struct {
	VisitID         uuid.UUID  `json:"visit_id"`
	ServiceTypeID   uuid.UUID  `json:"service_type_id"`
	WorkerID        *uuid.UUID `json:"worker_id"`
	AttendeeCount   int        `json:"attendee_count"`
	SessionDate     string     `json:"session_date"`
	SessionTime     *string    `json:"session_time"`
	DurationMinutes *int       `json:"duration_minutes"`
	Notes           *string    `json:"notes"`
}

// ── Service ──────────────────────────────────────────────────────────────────

type Service struct {
	repo caresession.Repository
}

func NewService(repo caresession.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateCareSessionRequest, createdBy uuid.UUID) (*caresession.CareSession, error) {
	sessionDate, err := time.Parse("2006-01-02", req.SessionDate)
	if err != nil {
		return nil, errors.New("invalid session_date format, expected YYYY-MM-DD")
	}

	sessionType := req.SessionType
	if sessionType == "" {
		if req.CompanyID != nil {
			sessionType = "company_visit"
		} else {
			sessionType = "particular"
		}
	}

	cs := &caresession.CareSession{
		VisitID:           req.VisitID,
		PatientID:         req.PatientID,
		WorkerID:          req.WorkerID,
		ServiceTypeID:     req.ServiceTypeID,
		CompanyID:         req.CompanyID,
		ContractServiceID: req.ContractServiceID,
		SessionType:       sessionType,
		SessionDate:       sessionDate,
		SessionTime:       req.SessionTime,
		DurationMinutes:   req.DurationMinutes,
		Status:            "completed",
		Notes:             req.Notes,
		FollowUpRequired:  false,
		CreatedBy:         &createdBy,
	}

	return cs, s.repo.Create(ctx, cs)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*caresession.CareSession, error) {
	cs, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return cs, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateCareSessionRequest, updatedBy uuid.UUID) (*caresession.CareSession, error) {
	cs, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	if req.Status != nil {
		cs.Status = *req.Status
	}
	if req.DurationMinutes != nil {
		cs.DurationMinutes = req.DurationMinutes
	}
	if req.ChiefComplaint != nil {
		cs.ChiefComplaint = req.ChiefComplaint
	}
	if req.Subjective != nil {
		cs.Subjective = req.Subjective
	}
	if req.Objective != nil {
		cs.Objective = req.Objective
	}
	if req.Assessment != nil {
		cs.Assessment = req.Assessment
	}
	if req.Plan != nil {
		cs.Plan = req.Plan
	}
	if req.Notes != nil {
		cs.Notes = req.Notes
	}
	if req.FollowUpRequired != nil {
		cs.FollowUpRequired = *req.FollowUpRequired
	}
	if req.FollowUpStatus != nil {
		cs.FollowUpStatus = req.FollowUpStatus
	}
	if req.FollowUpDate != nil {
		t, err := time.Parse("2006-01-02", *req.FollowUpDate)
		if err == nil {
			cs.FollowUpDate = &t
		}
	}
	if req.FollowUpNotes != nil {
		cs.FollowUpNotes = req.FollowUpNotes
	}
	cs.UpdatedBy = &updatedBy
	return cs, s.repo.Update(ctx, cs)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return ErrNotFound
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context, patientIDStr, workerIDStr, companyIDStr, visitIDStr, sessionType, status, dateFrom, dateTo string, limit, offset int) ([]*caresession.CareSession, int64, error) {
	f := caresession.Filter{
		SessionType: sessionType,
		Status:      status,
	}
	if patientIDStr != "" {
		if id, err := uuid.Parse(patientIDStr); err == nil {
			f.PatientID = &id
		}
	}
	if workerIDStr != "" {
		if id, err := uuid.Parse(workerIDStr); err == nil {
			f.WorkerID = &id
		}
	}
	if companyIDStr != "" {
		if id, err := uuid.Parse(companyIDStr); err == nil {
			f.CompanyID = &id
		}
	}
	if visitIDStr != "" {
		if id, err := uuid.Parse(visitIDStr); err == nil {
			f.VisitID = &id
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

func (s *Service) CreateGroupSession(ctx context.Context, req CreateGroupSessionRequest, createdBy uuid.UUID) (*caresession.GroupSession, error) {
	sessionDate, err := time.Parse("2006-01-02", req.SessionDate)
	if err != nil {
		return nil, errors.New("invalid session_date format")
	}
	gs := &caresession.GroupSession{
		VisitID:         req.VisitID,
		ServiceTypeID:   req.ServiceTypeID,
		WorkerID:        req.WorkerID,
		AttendeeCount:   req.AttendeeCount,
		SessionDate:     sessionDate,
		SessionTime:     req.SessionTime,
		DurationMinutes: req.DurationMinutes,
		Notes:           req.Notes,
		CreatedBy:       &createdBy,
	}
	return gs, s.repo.CreateGroupSession(ctx, gs)
}

func (s *Service) ListGroupSessions(ctx context.Context, visitID uuid.UUID) ([]*caresession.GroupSession, error) {
	return s.repo.ListGroupSessions(ctx, visitID)
}
