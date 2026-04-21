package appointment

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	StatusRequested  = "requested"
	StatusConfirmed  = "confirmed"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusCancelled  = "cancelled"
	StatusNoShow     = "no_show"
)

type Appointment struct {
	ID               uuid.UUID  `db:"id"                 json:"id"`
	PatientID        uuid.UUID  `db:"patient_id"         json:"patient_id"`
	WorkerID         *uuid.UUID `db:"worker_id"          json:"worker_id,omitempty"`
	ServiceTypeID    uuid.UUID  `db:"service_type_id"    json:"service_type_id"`
	CompanyID        *uuid.UUID `db:"company_id"         json:"company_id,omitempty"`
	RecurringGroupID *uuid.UUID `db:"recurring_group_id" json:"recurring_group_id,omitempty"`
	ScheduledAt      time.Time  `db:"scheduled_at"       json:"scheduled_at"`
	DurationMinutes  *int       `db:"duration_minutes"   json:"duration_minutes,omitempty"`
	Status           string     `db:"status"             json:"status"`
	Notes            *string    `db:"notes"              json:"notes,omitempty"`
	ChiefComplaint   *string    `db:"chief_complaint"    json:"chief_complaint,omitempty"`
	Subjective       *string    `db:"subjective"         json:"subjective,omitempty"`
	Objective        *string    `db:"objective"          json:"objective,omitempty"`
	Assessment       *string    `db:"assessment"         json:"assessment,omitempty"`
	Plan             *string    `db:"plan"               json:"plan,omitempty"`
	FollowUpRequired bool       `db:"follow_up_required" json:"follow_up_required"`
	FollowUpNotes    *string    `db:"follow_up_notes"    json:"follow_up_notes,omitempty"`
	FollowUpDate     *time.Time `db:"follow_up_date"     json:"follow_up_date,omitempty"`
	CareSessionID    *uuid.UUID `db:"care_session_id"    json:"care_session_id,omitempty"`
	CreatedAt        time.Time  `db:"created_at"         json:"created_at"`
	UpdatedAt        *time.Time `db:"updated_at"         json:"updated_at,omitempty"`
	CreatedBy        *uuid.UUID `db:"created_by"         json:"created_by,omitempty"`

	// Enrichments (populated via JOIN, not real columns)
	PatientName     *string `db:"patient_name"      json:"patient_name,omitempty"`
	WorkerName      *string `db:"worker_name"       json:"worker_name,omitempty"`
	ServiceTypeName *string `db:"service_type_name" json:"service_type_name,omitempty"`
	CompanyName     *string `db:"company_name"      json:"company_name,omitempty"`
}

type Filter struct {
	PatientID *uuid.UUID
	WorkerID  *uuid.UUID
	CompanyID *uuid.UUID
	Status    string
	DateFrom  *time.Time
	DateTo    *time.Time
}

type Repository interface {
	Create(ctx context.Context, a *Appointment) error
	CreateBatch(ctx context.Context, batch []*Appointment) error
	FindByID(ctx context.Context, id uuid.UUID) (*Appointment, error)
	Update(ctx context.Context, a *Appointment) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f Filter, limit, offset int) ([]*Appointment, int64, error)
	HasWorkerConflict(ctx context.Context, workerID uuid.UUID, scheduledAt time.Time, durationMinutes int, excludeID *uuid.UUID) (bool, error)
}
