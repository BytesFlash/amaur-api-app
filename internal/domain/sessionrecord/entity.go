package sessionrecord

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SessionRecord registra la evolución clínica de una sesión realizada.
type SessionRecord struct {
	ID                  uuid.UUID  `json:"id"`
	TreatmentPlanID     uuid.UUID  `json:"treatment_plan_id"`
	AppointmentID       *uuid.UUID `json:"appointment_id,omitempty"`
	PatientID           uuid.UUID  `json:"patient_id"`
	ProfessionalID      uuid.UUID  `json:"professional_id"`
	SessionNumber       int        `json:"session_number"`
	EvolutionNotes      *string    `json:"evolution_notes,omitempty"`
	PerformedTreatment  *string    `json:"performed_treatment,omitempty"`
	PatientInstructions *string    `json:"patient_instructions,omitempty"`
	PainLevel           *int       `json:"pain_level,omitempty"`
	NextAction          *string    `json:"next_action,omitempty"`
	FollowUpRequired    bool       `json:"follow_up_required"`
	FollowUpDate        *time.Time `json:"follow_up_date,omitempty"`
	InternalNotes       *string    `json:"internal_notes,omitempty"`
	CreatedBy           *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           *time.Time `json:"updated_at,omitempty"`

	// Enriquecimientos
	ProfessionalName *string `json:"professional_name,omitempty"`
}

// Repository define las operaciones de persistencia del dominio.
type Repository interface {
	Create(ctx context.Context, record *SessionRecord) error
	FindByID(ctx context.Context, id uuid.UUID) (*SessionRecord, error)
	Update(ctx context.Context, record *SessionRecord) error
	ListByPlan(ctx context.Context, treatmentPlanID uuid.UUID) ([]*SessionRecord, error)
	FindByAppointment(ctx context.Context, appointmentID uuid.UUID) (*SessionRecord, error)
}
