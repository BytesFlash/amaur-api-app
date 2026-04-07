package appointment

import (
	"context"
	"time"

	"github.com/google/uuid"
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
	CareSessionID    *uuid.UUID `db:"care_session_id"    json:"care_session_id,omitempty"`
	CreatedAt        time.Time  `db:"created_at"         json:"created_at"`
	UpdatedAt        *time.Time `db:"updated_at"         json:"updated_at,omitempty"`
	CreatedBy        *uuid.UUID `db:"created_by"         json:"created_by,omitempty"`

	// Enrichments (populated via JOIN, not real columns)
	PatientName     *string `db:"patient_name"      json:"patient_name,omitempty"`
	WorkerName      *string `db:"worker_name"       json:"worker_name,omitempty"`
	ServiceTypeName *string `db:"service_type_name" json:"service_type_name,omitempty"`
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
