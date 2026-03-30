package visit

import (
	"context"
	"errors"
	"time"

	"amaur/api/internal/domain/visit"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("visit not found")

type CreateVisitRequest struct {
	CompanyID         uuid.UUID   `json:"company_id"`
	BranchID          *uuid.UUID  `json:"branch_id"`
	ContractID        *uuid.UUID  `json:"contract_id"`
	ScheduledDate     string      `json:"scheduled_date"`
	ScheduledStart    *string     `json:"scheduled_start"`
	ScheduledEnd      *string     `json:"scheduled_end"`
	CoordinatorUserID *uuid.UUID  `json:"coordinator_user_id"`
	GeneralNotes      *string     `json:"general_notes"`
	WorkerIDs         []uuid.UUID `json:"worker_ids"`
}

type UpdateVisitRequest struct {
	Status             *string `json:"status"`
	ScheduledDate      *string `json:"scheduled_date"`
	ScheduledStart     *string `json:"scheduled_start"`
	ScheduledEnd       *string `json:"scheduled_end"`
	ActualStart        *string `json:"actual_start"`
	ActualEnd          *string `json:"actual_end"`
	GeneralNotes       *string `json:"general_notes"`
	CancellationReason *string `json:"cancellation_reason"`
	InternalReport     *string `json:"internal_report"`
}

type Service struct {
	repo visit.Repository
}

func NewService(repo visit.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateVisitRequest) (*visit.Visit, error) {
	scheduledDate, err := parseDate(req.ScheduledDate)
	if err != nil {
		return nil, errors.New("scheduled_date inválido, use formato YYYY-MM-DD")
	}
	v := &visit.Visit{
		CompanyID:         req.CompanyID,
		BranchID:          req.BranchID,
		ContractID:        req.ContractID,
		Status:            "scheduled",
		ScheduledDate:     scheduledDate,
		ScheduledStart:    req.ScheduledStart,
		ScheduledEnd:      req.ScheduledEnd,
		CoordinatorUserID: req.CoordinatorUserID,
		GeneralNotes:      req.GeneralNotes,
	}
	if err := s.repo.Create(ctx, v); err != nil {
		return nil, err
	}
	if len(req.WorkerIDs) > 0 {
		_ = s.repo.AssignWorkers(ctx, v.ID, req.WorkerIDs)
	}
	return v, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*visit.Visit, error) {
	v, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	workers, err := s.repo.ListWorkers(ctx, id)
	if err == nil {
		v.Workers = workers
	}
	return v, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateVisitRequest) (*visit.Visit, error) {
	v, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	if req.Status != nil {
		v.Status = *req.Status
	}
	if req.ScheduledDate != nil {
		if t, err := parseDate(*req.ScheduledDate); err == nil {
			v.ScheduledDate = t
		}
	}
	if req.ScheduledStart != nil {
		v.ScheduledStart = req.ScheduledStart
	}
	if req.ScheduledEnd != nil {
		v.ScheduledEnd = req.ScheduledEnd
	}
	if req.ActualStart != nil {
		if t, err := parseDateTime(*req.ActualStart); err == nil {
			v.ActualStart = &t
		}
	}
	if req.ActualEnd != nil {
		if t, err := parseDateTime(*req.ActualEnd); err == nil {
			v.ActualEnd = &t
		}
	}
	if req.GeneralNotes != nil {
		v.GeneralNotes = req.GeneralNotes
	}
	if req.CancellationReason != nil {
		v.CancellationReason = req.CancellationReason
	}
	if req.InternalReport != nil {
		v.InternalReport = req.InternalReport
	}
	if err := s.repo.Update(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return ErrNotFound
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context, companyIDStr, patientIDStr, status, dateFrom, dateTo string, limit, offset int) ([]*visit.Visit, int64, error) {
	f := visit.Filter{}
	if companyIDStr != "" {
		if id, err := uuid.Parse(companyIDStr); err == nil {
			f.CompanyID = &id
		}
	}
	if patientIDStr != "" {
		if id, err := uuid.Parse(patientIDStr); err == nil {
			f.PatientID = &id
		}
	}
	f.Status = status
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

func (s *Service) HasPatientParticipation(ctx context.Context, visitID, patientID uuid.UUID) (bool, error) {
	return s.repo.HasPatientParticipation(ctx, visitID, patientID)
}

func parseDate(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

func parseDateTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}
