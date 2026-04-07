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
	Attended        bool       `db:"attended" json:"attended"`
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
	ScheduledDate   time.Time `db:"scheduled_date" json:"scheduled_date"`
	ServiceTypeName *string   `db:"service_type_name" json:"service_type_name,omitempty"`
}

type Filter struct {
	CompanyID  *uuid.UUID
	ContractID *uuid.UUID
	Status     ProgramStatus
	DateFrom   *time.Time
	DateTo     *time.Time
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
}
