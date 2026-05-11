package program

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ProgramStatus string

const (
	StatusDraft     ProgramStatus = "draft"
	StatusActive    ProgramStatus = "active"
	StatusCompleted ProgramStatus = "completed"
	StatusCancelled ProgramStatus = "cancelled"
)

type CompanyProgram struct {
	ID         uuid.UUID     `db:"id" json:"id"`
	CompanyID  uuid.UUID     `db:"company_id" json:"company_id"`
	ContractID uuid.UUID     `db:"contract_id" json:"contract_id"`
	Name       string        `db:"name" json:"name"`
	StartDate  time.Time     `db:"start_date" json:"start_date"`
	EndDate    *time.Time    `db:"end_date" json:"end_date,omitempty"`
	Status     ProgramStatus `db:"status" json:"status"`
	Notes      *string       `db:"notes" json:"notes,omitempty"`
	CreatedAt  time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt  *time.Time    `db:"updated_at" json:"updated_at,omitempty"`
	CreatedBy  *uuid.UUID    `db:"created_by" json:"created_by,omitempty"`
	UpdatedBy  *uuid.UUID    `db:"updated_by" json:"updated_by,omitempty"`
	// Enrichment (populated by JOIN in list queries)
	CompanyName *string `db:"company_name" json:"company_name,omitempty"`
}

type ScheduleRule struct {
	ID                     uuid.UUID  `db:"id" json:"id"`
	ProgramID              uuid.UUID  `db:"program_id" json:"program_id"`
	Weekday                int16      `db:"weekday" json:"weekday"`
	StartTime              string     `db:"start_time" json:"start_time"`
	DurationMinutes        int        `db:"duration_minutes" json:"duration_minutes"`
	FrequencyIntervalWeeks int        `db:"frequency_interval_weeks" json:"frequency_interval_weeks"`
	MaxOccurrences         *int       `db:"max_occurrences" json:"max_occurrences,omitempty"`
	ServiceTypeID          *uuid.UUID `db:"service_type_id" json:"service_type_id,omitempty"`
	WorkerID               *uuid.UUID `db:"worker_id" json:"worker_id,omitempty"`
	CreatedAt              time.Time  `db:"created_at" json:"created_at"`
	CreatedBy              *uuid.UUID `db:"created_by" json:"created_by,omitempty"`
}

type AgendaServiceStatus string

const (
	AgendaServicePlanned   AgendaServiceStatus = "planned"
	AgendaServiceCompleted AgendaServiceStatus = "completed"
	AgendaServiceCancelled AgendaServiceStatus = "cancelled"
)

type AgendaService struct {
	ID                     uuid.UUID           `db:"id" json:"id"`
	AgendaID               uuid.UUID           `db:"agenda_id" json:"agenda_id"`
	ServiceTypeID          uuid.UUID           `db:"service_type_id" json:"service_type_id"`
	WorkerID               *uuid.UUID          `db:"worker_id" json:"worker_id,omitempty"`
	PlannedStartTime       *string             `db:"planned_start_time" json:"planned_start_time,omitempty"`
	PlannedDurationMinutes *int                `db:"planned_duration_minutes" json:"planned_duration_minutes,omitempty"`
	Status                 AgendaServiceStatus `db:"status" json:"status"`
	Notes                  *string             `db:"notes" json:"notes,omitempty"`
	CompletedAt            *time.Time          `db:"completed_at" json:"completed_at,omitempty"`
	CompletedBy            *uuid.UUID          `db:"completed_by" json:"completed_by,omitempty"`
	CreatedAt              time.Time           `db:"created_at" json:"created_at"`
	UpdatedAt              *time.Time          `db:"updated_at" json:"updated_at,omitempty"`
}

type AgendaServiceParticipant struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	AgendaServiceID uuid.UUID  `db:"agenda_service_id" json:"agenda_service_id"`
	PatientID       uuid.UUID  `db:"patient_id" json:"patient_id"`
	Attended        *bool      `db:"attended" json:"attended"`
	AttendedAt      *time.Time `db:"attended_at" json:"attended_at,omitempty"`
	Notes           *string    `db:"notes" json:"notes,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	CreatedBy       *uuid.UUID `db:"created_by" json:"created_by,omitempty"`
}

type AgendaServiceContext struct {
	AgendaID       uuid.UUID `db:"agenda_id" json:"agenda_id"`
	CompanyID      uuid.UUID `db:"company_id" json:"company_id"`
	ScheduledDate  time.Time `db:"scheduled_date" json:"scheduled_date"`
	ScheduledStart *string   `db:"scheduled_start" json:"scheduled_start,omitempty"`
}

// AgendaServiceDetail is AgendaService enriched with display names.
type AgendaServiceDetail struct {
	AgendaService
	ServiceTypeName *string `db:"service_type_name" json:"service_type_name,omitempty"`
	WorkerName      *string `db:"worker_name"      json:"worker_name,omitempty"`
}

// AgendaWithServices is an agenda row enriched with its agenda_services list.
type AgendaWithServices struct {
	AgendaID       uuid.UUID              `db:"agenda_id"       json:"id"`
	ScheduledDate  time.Time              `db:"scheduled_date"  json:"scheduled_date"`
	ScheduledStart *string                `db:"scheduled_start" json:"scheduled_start,omitempty"`
	Status         string                 `db:"status"          json:"status"`
	Services       []*AgendaServiceDetail `db:"-"               json:"services"`
}

// ParticipantDetail is AgendaServiceParticipant enriched with patient name.
type ParticipantDetail struct {
	AgendaServiceParticipant
	PatientName *string `db:"patient_name" json:"patient_name,omitempty"`
}

// AgendaServiceWithDate is an agenda_services row enriched with the agenda's scheduled_date.
type AgendaServiceWithDate struct {
	AgendaService
	ScheduledDate    time.Time `db:"scheduled_date"    json:"scheduled_date"`
	ServiceTypeName  *string   `db:"service_type_name" json:"service_type_name,omitempty"`
	ProgramName      *string   `db:"program_name"      json:"program_name,omitempty"`
	CompanyName      *string   `db:"company_name"      json:"company_name,omitempty"`
	ParticipantCount *int      `db:"participant_count" json:"participant_count,omitempty"`
}

type Filter struct {
	CompanyID  *uuid.UUID
	ContractID *uuid.UUID
	WorkerID   *uuid.UUID
	Status     ProgramStatus
	DateFrom   *time.Time
	DateTo     *time.Time
}

// PatientProgramParticipation represents a single group session a patient participated (or was enrolled) in.
type PatientProgramParticipation struct {
	ProgramID       uuid.UUID           `json:"program_id"`
	ProgramName     string              `json:"program_name"`
	CompanyID       uuid.UUID           `json:"company_id"`
	AgendaID        uuid.UUID           `json:"agenda_id"`
	ScheduledDate   time.Time           `json:"scheduled_date"`
	AgendaServiceID uuid.UUID           `json:"agenda_service_id"`
	ServiceTypeName string              `json:"service_type_name"`
	WorkerName      *string             `json:"worker_name,omitempty"`
	PlannedStartTime *string            `json:"planned_start_time,omitempty"`
	DurationMinutes  *int               `json:"duration_minutes,omitempty"`
	ServiceStatus   AgendaServiceStatus `json:"service_status"`
	Attended        *bool               `json:"attended"`
	AttendedAt      *time.Time          `json:"attended_at,omitempty"`
	Notes           *string             `json:"notes,omitempty"`
}

type Repository interface {
	CreateProgram(ctx context.Context, p *CompanyProgram) error
	GetProgramByID(ctx context.Context, id uuid.UUID) (*CompanyProgram, error)
	UpdateProgram(ctx context.Context, p *CompanyProgram) error
	ListPrograms(ctx context.Context, f Filter, limit, offset int) ([]*CompanyProgram, int64, error)

	CreateScheduleRules(ctx context.Context, rules []*ScheduleRule) error
	ListScheduleRules(ctx context.Context, programID uuid.UUID) ([]*ScheduleRule, error)
	ReplaceScheduleRules(ctx context.Context, programID uuid.UUID, rules []*ScheduleRule) error

	CreateAgendaServices(ctx context.Context, services []*AgendaService) error
	ListAgendaServices(ctx context.Context, agendaID uuid.UUID) ([]*AgendaService, error)
	GetAgendaServiceByID(ctx context.Context, id uuid.UUID) (*AgendaService, error)
	GetAgendaContextByServiceID(ctx context.Context, agendaServiceID uuid.UUID) (*AgendaServiceContext, error)
	GetAgendaContextByAgendaID(ctx context.Context, agendaID uuid.UUID) (*AgendaServiceContext, error)
	UpdateAgendaService(ctx context.Context, service *AgendaService) error

	UpsertParticipants(ctx context.Context, participants []*AgendaServiceParticipant) error
	ListParticipants(ctx context.Context, agendaServiceID uuid.UUID) ([]*AgendaServiceParticipant, error)
	ListParticipantsDetail(ctx context.Context, agendaServiceID uuid.UUID) ([]*ParticipantDetail, error)
	PatientIDsOutsideAgendaCompany(ctx context.Context, agendaServiceID uuid.UUID, patientIDs []uuid.UUID) ([]uuid.UUID, error)

	LinkProgramAgenda(ctx context.Context, programID, agendaID uuid.UUID) error
	ListProgramAgendas(ctx context.Context, programID uuid.UUID) ([]*AgendaWithServices, error)
	// CreateAgenda creates a row in the agendas table and returns the new id.
	CreateAgenda(ctx context.Context, companyID uuid.UUID, contractID *uuid.UUID, scheduledDate time.Time, scheduledStart *string, byUserID uuid.UUID) (uuid.UUID, error)
	// ListAgendaServicesByWorker returns agenda services assigned to a worker within [from, to).
	ListAgendaServicesByWorker(ctx context.Context, workerID uuid.UUID, from, to time.Time) ([]*AgendaServiceWithDate, error)
	ListCompanyPatientIDs(ctx context.Context, companyID uuid.UUID) ([]uuid.UUID, error)
	HasWorkerScheduleConflict(ctx context.Context, workerID uuid.UUID, scheduledDate time.Time, startTime string, durationMinutes int, excludeAgendaServiceID *uuid.UUID) (bool, error)

	// ListPatientParticipation returns all group agenda service participations for a given patient,
	// enriched with program, agenda and service type information, ordered by scheduled date desc.
	ListPatientParticipation(ctx context.Context, patientID uuid.UUID) ([]*PatientProgramParticipation, error)

	// IsWorkerLinkedToProgram returns true if workerID appears in the program's schedule rules
	// or in any of its agenda services. Used for access checks to keep them in sync with
	// the worker filter applied by ListPrograms.
	IsWorkerLinkedToProgram(ctx context.Context, programID, workerID uuid.UUID) (bool, error)

	// ClearPendingAgendas removes all non-completed agendas linked to the program
	// (cascading through agenda_services and agenda_service_participants).
	// It only touches agendas whose status is 'scheduled' and that have no
	// agenda_service with status='completed'. Returns the number of agendas deleted.
	ClearPendingAgendas(ctx context.Context, programID uuid.UUID) (int, error)

	// DeleteProgram hard-deletes the program and all its associated data
	// (schedule rules, pending agendas, links). Returns ErrProgramHasCompletedSessions
	// if any agenda service linked to the program is already completed.
	DeleteProgram(ctx context.Context, programID uuid.UUID) error

	// ListWorkerGroupSessionHistory returns all agenda_services assigned to workerID,
	// ordered by scheduled date descending. Used to display the worker's group session history.
	ListWorkerGroupSessionHistory(ctx context.Context, workerID uuid.UUID) ([]*AgendaServiceWithDate, error)

	// Transact executes fn within a single DB transaction. A new Repository backed
	// by the transaction is passed to fn; all operations inside fn share the same tx.
	// The transaction is committed if fn returns nil, rolled back otherwise.
	Transact(ctx context.Context, fn func(tx Repository) error) error
}
