package followuptask

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusDone       = "done"
	StatusCancelled  = "cancelled"

	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"
)

// FollowUpTask representa una tarea de seguimiento asociada a un paciente.
type FollowUpTask struct {
	ID              uuid.UUID  `json:"id"`
	PatientID       uuid.UUID  `json:"patient_id"`
	TreatmentPlanID *uuid.UUID `json:"treatment_plan_id,omitempty"`
	AppointmentID   *uuid.UUID `json:"appointment_id,omitempty"`
	ProfessionalID  *uuid.UUID `json:"professional_id,omitempty"`
	Title           string     `json:"title"`
	Description     *string    `json:"description,omitempty"`
	DueDate         time.Time  `json:"due_date"`
	Status          string     `json:"status"`
	Priority        string     `json:"priority"`
	CreatedBy       *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`

	// Enriquecimientos
	PatientName      *string `json:"patient_name,omitempty"`
	ProfessionalName *string `json:"professional_name,omitempty"`
}

// Filter para listar tareas de seguimiento.
type Filter struct {
	PatientID      *uuid.UUID
	ProfessionalID *uuid.UUID
	Status         string
	DueBefore      *time.Time
}

// Repository define las operaciones de persistencia del dominio.
type Repository interface {
	Create(ctx context.Context, task *FollowUpTask) error
	FindByID(ctx context.Context, id uuid.UUID) (*FollowUpTask, error)
	Update(ctx context.Context, task *FollowUpTask) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f Filter, limit, offset int) ([]*FollowUpTask, int64, error)
}
