package worker

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SpecialtyItem is a row from the specialties catalog.
type SpecialtyItem struct {
	Code string `db:"code" json:"code"`
	Name string `db:"name" json:"name"`
}

// ApptSummary is a lightweight view of an appointment used in calendar responses.
type ApptSummary struct {
	ScheduledAt     string `json:"scheduled_at"` // "HH:MM"
	DurationMinutes int    `json:"duration_minutes"`
	Type            string `json:"type"`  // "individual" | "group"
	Label           string `json:"label"` // patient/company name or service type
}

// DayCalendar summarises availability and bookings for a single calendar day.
type DayCalendar struct {
	Date             string        `json:"date"`              // "YYYY-MM-DD"
	TotalMinutes     int           `json:"total_minutes"`     // capacity from availability rules
	AvailableMinutes int           `json:"available_minutes"` // total - booked (≥0)
	BookedMinutes    int           `json:"booked_minutes"`    // from confirmed/scheduled appointments
	Appointments     []ApptSummary `json:"appointments"`
}

// AvailabilityRule is a recurring weekly time block when a worker is available.
type AvailabilityRule struct {
	ID        uuid.UUID  `db:"id"         json:"id"`
	WorkerID  uuid.UUID  `db:"worker_id"  json:"worker_id"`
	Weekday   int16      `db:"weekday"    json:"weekday"`    // 0=Sunday … 6=Saturday
	StartTime string     `db:"start_time" json:"start_time"` // "HH:MM"
	EndTime   string     `db:"end_time"   json:"end_time"`   // "HH:MM"
	IsActive  bool       `db:"is_active"  json:"is_active"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	CreatedBy *uuid.UUID `db:"created_by" json:"created_by,omitempty"`
}

// TimeSlot represents a bookable time block for a worker in the slots endpoint.
type TimeSlot struct {
	Date      string `json:"date"` // "YYYY-MM-DD"
	Weekday   int    `json:"weekday"`
	StartTime string `json:"start_time"` // "HH:MM"
	EndTime   string `json:"end_time"`   // "HH:MM"
	Available bool   `json:"available"`
}

type Worker struct {
	ID                uuid.UUID  `db:"id" json:"id"`
	UserID            *uuid.UUID `db:"user_id" json:"user_id,omitempty"`
	RUT               *string    `db:"rut" json:"rut,omitempty"`
	FirstName         string     `db:"first_name" json:"first_name"`
	LastName          string     `db:"last_name" json:"last_name"`
	Email             *string    `db:"email" json:"email,omitempty"`
	Phone             *string    `db:"phone" json:"phone,omitempty"`
	RoleTitle         *string    `db:"role_title" json:"role_title,omitempty"`
	Specialty         *string    `db:"specialty" json:"specialty,omitempty"`
	HireDate          *time.Time `db:"hire_date" json:"hire_date,omitempty"`
	BirthDate         *time.Time `db:"birth_date" json:"birth_date,omitempty"`
	TerminationDate   *time.Time `db:"termination_date" json:"termination_date,omitempty"`
	IsActive          bool       `db:"is_active" json:"is_active"`
	AvailabilityNotes *string    `db:"availability_notes" json:"availability_notes,omitempty"`
	InternalNotes     *string    `db:"internal_notes" json:"internal_notes,omitempty"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt         *time.Time `db:"updated_at" json:"updated_at,omitempty"`
	CreatedBy         *uuid.UUID `db:"created_by" json:"created_by,omitempty"`
	UpdatedBy         *uuid.UUID `db:"updated_by" json:"updated_by,omitempty"`
	DeletedAt         *time.Time `db:"deleted_at" json:"-"`

	// Populated from worker_specialties join (not a DB column).
	Specialties []SpecialtyItem `db:"-" json:"specialties,omitempty"`
}

func (w *Worker) FullName() string {
	return w.FirstName + " " + w.LastName
}

type Repository interface {
	Create(ctx context.Context, w *Worker) error
	FindByID(ctx context.Context, id uuid.UUID) (*Worker, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) (*Worker, error)
	Update(ctx context.Context, w *Worker) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	ListActive(ctx context.Context, limit, offset int) ([]*Worker, int64, error)
	// specialtyCode filters by specialty (empty string = no filter).
	List(ctx context.Context, search string, specialtyCode string, onlyActive bool, limit, offset int) ([]*Worker, int64, error)
	LinkUser(ctx context.Context, workerID, userID uuid.UUID) error

	// Specialty catalog
	ListSpecialties(ctx context.Context) ([]SpecialtyItem, error)
	// CreateSpecialty adds a new entry to the specialty catalog.
	CreateSpecialty(ctx context.Context, item SpecialtyItem) error
	// DeleteSpecialty removes a specialty from the catalog (fails if in use).
	DeleteSpecialty(ctx context.Context, code string) error
	// GetWorkerSpecialties returns the specialties linked to a specific worker.
	GetWorkerSpecialties(ctx context.Context, workerID uuid.UUID) ([]SpecialtyItem, error)
	// SetWorkerSpecialties replaces all specialty links for a worker atomically.
	SetWorkerSpecialties(ctx context.Context, workerID uuid.UUID, codes []string, setBy uuid.UUID) error

	// Availability schedule
	ListAvailabilityRules(ctx context.Context, workerID uuid.UUID) ([]*AvailabilityRule, error)
	ReplaceAvailabilityRules(ctx context.Context, workerID uuid.UUID, rules []*AvailabilityRule) error
}
