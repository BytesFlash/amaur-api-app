package caresession

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CareSession represents an individual clinical/wellness session with a patient.
type CareSession struct {
	ID                uuid.UUID  `db:"id"                  json:"id"`
	VisitID           *uuid.UUID `db:"visit_id"            json:"visit_id,omitempty"`
	PatientID         uuid.UUID  `db:"patient_id"          json:"patient_id"`
	WorkerID          uuid.UUID  `db:"worker_id"           json:"worker_id"`
	ServiceTypeID     uuid.UUID  `db:"service_type_id"     json:"service_type_id"`
	CompanyID         *uuid.UUID `db:"company_id"          json:"company_id,omitempty"`
	ContractServiceID *uuid.UUID `db:"contract_service_id" json:"contract_service_id,omitempty"`
	SessionType       string     `db:"session_type"        json:"session_type"` // company_visit | particular
	SessionDate       time.Time  `db:"session_date"        json:"session_date"`
	SessionTime       *string    `db:"session_time"        json:"session_time,omitempty"`
	DurationMinutes   *int       `db:"duration_minutes"    json:"duration_minutes,omitempty"`
	Status            string     `db:"status"              json:"status"`
	// SOAP notes
	ChiefComplaint *string `db:"chief_complaint" json:"chief_complaint,omitempty"`
	Subjective     *string `db:"subjective"      json:"subjective,omitempty"`
	Objective      *string `db:"objective"       json:"objective,omitempty"`
	Assessment     *string `db:"assessment"      json:"assessment,omitempty"`
	Plan           *string `db:"plan"            json:"plan,omitempty"`
	Notes          *string `db:"notes"           json:"notes,omitempty"`
	// Follow-up
	FollowUpRequired    bool       `db:"follow_up_required"       json:"follow_up_required"`
	FollowUpStatus      *string    `db:"follow_up_status"         json:"follow_up_status,omitempty"`
	FollowUpDate        *time.Time `db:"follow_up_date"           json:"follow_up_date,omitempty"`
	FollowUpNotes       *string    `db:"follow_up_notes"          json:"follow_up_notes,omitempty"`
	FollowUpContactedAt *time.Time `db:"follow_up_contacted_at"   json:"follow_up_contacted_at,omitempty"`
	// Audit
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at,omitempty"`
	CreatedBy *uuid.UUID `db:"created_by" json:"-"`
	UpdatedBy *uuid.UUID `db:"updated_by" json:"-"`
	// Enrichment (not in DB, populated by JOIN)
	PatientFirstName *string `db:"patient_first_name" json:"patient_first_name,omitempty"`
	PatientLastName  *string `db:"patient_last_name"  json:"patient_last_name,omitempty"`
	WorkerFirstName  *string `db:"worker_first_name"  json:"worker_first_name,omitempty"`
	WorkerLastName   *string `db:"worker_last_name"   json:"worker_last_name,omitempty"`
	ServiceTypeName  *string `db:"service_type_name"  json:"service_type_name,omitempty"`
	CompanyName      *string `db:"company_name"       json:"company_name,omitempty"`
}

// GroupSession tracks a group wellness activity (e.g. pausa activa).
type GroupSession struct {
	ID              uuid.UUID  `db:"id"               json:"id"`
	VisitID         uuid.UUID  `db:"visit_id"         json:"visit_id"`
	ServiceTypeID   uuid.UUID  `db:"service_type_id"  json:"service_type_id"`
	WorkerID        *uuid.UUID `db:"worker_id"        json:"worker_id,omitempty"`
	AttendeeCount   int        `db:"attendee_count"   json:"attendee_count"`
	SessionDate     time.Time  `db:"session_date"     json:"session_date"`
	SessionTime     *string    `db:"session_time"     json:"session_time,omitempty"`
	DurationMinutes *int       `db:"duration_minutes" json:"duration_minutes,omitempty"`
	Notes           *string    `db:"notes"            json:"notes,omitempty"`
	CreatedAt       time.Time  `db:"created_at"       json:"created_at"`
	CreatedBy       *uuid.UUID `db:"created_by"       json:"-"`
	// Enrichment
	ServiceTypeName *string `db:"service_type_name" json:"service_type_name,omitempty"`
	WorkerFirstName *string `db:"worker_first_name" json:"worker_first_name,omitempty"`
	WorkerLastName  *string `db:"worker_last_name"  json:"worker_last_name,omitempty"`
}

// Filter for querying care sessions.
type Filter struct {
	PatientID   *uuid.UUID
	WorkerID    *uuid.UUID
	CompanyID   *uuid.UUID
	VisitID     *uuid.UUID
	SessionType string
	Status      string
	DateFrom    *time.Time
	DateTo      *time.Time
}

// Repository defines the persistence contract for care and group sessions.
type Repository interface {
	Create(ctx context.Context, cs *CareSession) error
	FindByID(ctx context.Context, id uuid.UUID) (*CareSession, error)
	Update(ctx context.Context, cs *CareSession) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f Filter, limit, offset int) ([]*CareSession, int64, error)

	CreateGroupSession(ctx context.Context, gs *GroupSession) error
	ListGroupSessions(ctx context.Context, visitID uuid.UUID) ([]*GroupSession, error)
}
