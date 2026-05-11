package followuptask

import (
	"context"
	"errors"
	"time"

	"amaur/api/internal/domain/followuptask"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("follow-up task not found")
	ErrInvalidStatus = errors.New("invalid follow-up task status")
)

type CreateRequest struct {
	PatientID       uuid.UUID  `json:"patient_id"`
	TreatmentPlanID *uuid.UUID `json:"treatment_plan_id"`
	AppointmentID   *uuid.UUID `json:"appointment_id"`
	ProfessionalID  *uuid.UUID `json:"professional_id"`
	Title           string     `json:"title"`
	Description     *string    `json:"description"`
	DueDate         string     `json:"due_date"` // YYYY-MM-DD
	Priority        string     `json:"priority"`
}

type UpdateRequest struct {
	Title          *string    `json:"title"`
	Description    *string    `json:"description"`
	DueDate        *string    `json:"due_date"`
	Status         *string    `json:"status"`
	Priority       *string    `json:"priority"`
	ProfessionalID *uuid.UUID `json:"professional_id"`
}

type Service struct {
	repo followuptask.Repository
}

func NewService(repo followuptask.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateRequest, by uuid.UUID) (*followuptask.FollowUpTask, error) {
	due, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		return nil, errors.New("invalid due_date, expected YYYY-MM-DD")
	}
	priority := req.Priority
	if priority == "" {
		priority = followuptask.PriorityMedium
	}
	t := &followuptask.FollowUpTask{
		ID:              uuid.New(),
		PatientID:       req.PatientID,
		TreatmentPlanID: req.TreatmentPlanID,
		AppointmentID:   req.AppointmentID,
		ProfessionalID:  req.ProfessionalID,
		Title:           req.Title,
		Description:     req.Description,
		DueDate:         due,
		Status:          followuptask.StatusPending,
		Priority:        priority,
		CreatedBy:       &by,
		CreatedAt:       time.Now(),
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, t.ID)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*followuptask.FollowUpTask, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return t, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*followuptask.FollowUpTask, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	if req.Title != nil {
		t.Title = *req.Title
	}
	if req.Description != nil {
		t.Description = req.Description
	}
	if req.DueDate != nil {
		if due, err := time.Parse("2006-01-02", *req.DueDate); err == nil {
			t.DueDate = due
		}
	}
	if req.Status != nil {
		validStatuses := map[string]bool{
			followuptask.StatusPending:    true,
			followuptask.StatusInProgress: true,
			followuptask.StatusDone:       true,
			followuptask.StatusCancelled:  true,
		}
		if !validStatuses[*req.Status] {
			return nil, ErrInvalidStatus
		}
		t.Status = *req.Status
	}
	if req.Priority != nil {
		t.Priority = *req.Priority
	}
	if req.ProfessionalID != nil {
		t.ProfessionalID = req.ProfessionalID
	}
	if err := s.repo.Update(ctx, t); err != nil {
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

func (s *Service) List(ctx context.Context, patientID, professionalID, status, dueBefore string, limit, offset int) ([]*followuptask.FollowUpTask, int64, error) {
	f := followuptask.Filter{Status: status}
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
	if dueBefore != "" {
		if t, err := time.Parse("2006-01-02", dueBefore); err == nil {
			f.DueBefore = &t
		}
	}
	return s.repo.List(ctx, f, limit, offset)
}
