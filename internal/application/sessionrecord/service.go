package sessionrecord

import (
	"context"
	"errors"
	"time"

	"amaur/api/internal/domain/sessionrecord"
	"amaur/api/internal/domain/treatmentplan"

	"github.com/google/uuid"
)

var (
	ErrNotFound         = errors.New("session record not found")
	ErrPlanTerminal     = errors.New("treatment plan is already completed or cancelled")
	ErrNoProfessional   = errors.New("professional_id is required and no professional is assigned to the plan")
)

type CreateRequest struct {
	TreatmentPlanID     uuid.UUID  `json:"treatment_plan_id"`
	AppointmentID       *uuid.UUID `json:"appointment_id"`
	// ProfessionalID es opcional: si no se provee (o es UUID cero) se usa el profesional del plan.
	ProfessionalID      *uuid.UUID `json:"professional_id"`
	SessionNumber       int        `json:"session_number"`
	EvolutionNotes      *string    `json:"evolution_notes"`
	PerformedTreatment  *string    `json:"performed_treatment"`
	PatientInstructions *string    `json:"patient_instructions"`
	PainLevel           *int       `json:"pain_level"`
	NextAction          *string    `json:"next_action"`
	FollowUpRequired    bool       `json:"follow_up_required"`
	FollowUpDate        *string    `json:"follow_up_date"` // YYYY-MM-DD
	InternalNotes       *string    `json:"internal_notes"`
}

type UpdateRequest struct {
	EvolutionNotes      *string `json:"evolution_notes"`
	PerformedTreatment  *string `json:"performed_treatment"`
	PatientInstructions *string `json:"patient_instructions"`
	PainLevel           *int    `json:"pain_level"`
	NextAction          *string `json:"next_action"`
	FollowUpRequired    *bool   `json:"follow_up_required"`
	FollowUpDate        *string `json:"follow_up_date"`
	InternalNotes       *string `json:"internal_notes"`
}

type Service struct {
	repo      sessionrecord.Repository
	planRepo  treatmentplan.Repository
}

func NewService(repo sessionrecord.Repository, planRepo treatmentplan.Repository) *Service {
	return &Service{repo: repo, planRepo: planRepo}
}

func (s *Service) Create(ctx context.Context, req CreateRequest, by uuid.UUID) (*sessionrecord.SessionRecord, error) {
	plan, err := s.planRepo.FindByID(ctx, req.TreatmentPlanID)
	if err != nil {
		return nil, errors.New("treatment plan not found")
	}
	if plan.IsTerminal() {
		return nil, ErrPlanTerminal
	}

	// Resolver professional_id: usar el del request si es válido, sino el del plan.
	var professionalID uuid.UUID
	if req.ProfessionalID != nil && *req.ProfessionalID != uuid.Nil {
		professionalID = *req.ProfessionalID
	} else if plan.ProfessionalID != nil {
		professionalID = *plan.ProfessionalID
	} else {
		return nil, ErrNoProfessional
	}

	sr := &sessionrecord.SessionRecord{
		ID:                  uuid.New(),
		TreatmentPlanID:     req.TreatmentPlanID,
		AppointmentID:       req.AppointmentID,
		PatientID:           plan.PatientID,
		ProfessionalID:      professionalID,
		SessionNumber:       req.SessionNumber,
		EvolutionNotes:      req.EvolutionNotes,
		PerformedTreatment:  req.PerformedTreatment,
		PatientInstructions: req.PatientInstructions,
		PainLevel:           req.PainLevel,
		NextAction:          req.NextAction,
		FollowUpRequired:    req.FollowUpRequired,
		FollowUpDate:        parseDateOnly(req.FollowUpDate),
		InternalNotes:       req.InternalNotes,
		CreatedBy:           &by,
		CreatedAt:           time.Now(),
	}
	if err := s.repo.Create(ctx, sr); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, sr.ID)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*sessionrecord.SessionRecord, error) {
	sr, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return sr, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*sessionrecord.SessionRecord, error) {
	sr, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	if req.EvolutionNotes != nil {
		sr.EvolutionNotes = req.EvolutionNotes
	}
	if req.PerformedTreatment != nil {
		sr.PerformedTreatment = req.PerformedTreatment
	}
	if req.PatientInstructions != nil {
		sr.PatientInstructions = req.PatientInstructions
	}
	if req.PainLevel != nil {
		sr.PainLevel = req.PainLevel
	}
	if req.NextAction != nil {
		sr.NextAction = req.NextAction
	}
	if req.FollowUpRequired != nil {
		sr.FollowUpRequired = *req.FollowUpRequired
	}
	if req.FollowUpDate != nil {
		sr.FollowUpDate = parseDateOnly(req.FollowUpDate)
	}
	if req.InternalNotes != nil {
		sr.InternalNotes = req.InternalNotes
	}
	if err := s.repo.Update(ctx, sr); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, id)
}

func (s *Service) ListByPlan(ctx context.Context, planID uuid.UUID) ([]*sessionrecord.SessionRecord, error) {
	return s.repo.ListByPlan(ctx, planID)
}

func parseDateOnly(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	if t, err := time.Parse("2006-01-02", *s); err == nil {
		return &t
	}
	return nil
}
