package visit

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Visit struct {
	ID                 uuid.UUID  `db:"id"                   json:"id"`
	CompanyID          uuid.UUID  `db:"company_id"           json:"company_id"`
	BranchID           *uuid.UUID `db:"branch_id"            json:"branch_id,omitempty"`
	ContractID         *uuid.UUID `db:"contract_id"          json:"contract_id,omitempty"`
	Status             string     `db:"status"               json:"status"`
	ScheduledDate      time.Time  `db:"scheduled_date"       json:"scheduled_date"`
	ScheduledStart     *string    `db:"scheduled_start"      json:"scheduled_start,omitempty"`
	ScheduledEnd       *string    `db:"scheduled_end"        json:"scheduled_end,omitempty"`
	ActualStart        *time.Time `db:"actual_start"         json:"actual_start,omitempty"`
	ActualEnd          *time.Time `db:"actual_end"           json:"actual_end,omitempty"`
	CoordinatorUserID  *uuid.UUID `db:"coordinator_user_id"  json:"coordinator_user_id,omitempty"`
	GeneralNotes       *string    `db:"general_notes"        json:"general_notes,omitempty"`
	CancellationReason *string    `db:"cancellation_reason"  json:"cancellation_reason,omitempty"`
	InternalReport     *string    `db:"internal_report"      json:"internal_report,omitempty"`
	CreatedAt          time.Time  `db:"created_at"           json:"created_at"`
	UpdatedAt          *time.Time `db:"updated_at"           json:"updated_at,omitempty"`
	CreatedBy          *uuid.UUID `db:"created_by"           json:"-"`
	UpdatedBy          *uuid.UUID `db:"updated_by"           json:"-"`
	// Enrichment (populated by JOIN, not stored)
	CompanyName        *string        `db:"company_name"         json:"company_name,omitempty"`
	CompanyFantasyName *string        `db:"company_fantasy_name" json:"company_fantasy_name,omitempty"`
	Workers            []*VisitWorker `db:"-"                    json:"workers,omitempty"`
}

type VisitWorker struct {
	VisitID     uuid.UUID `db:"visit_id"    json:"visit_id"`
	WorkerID    uuid.UUID `db:"worker_id"   json:"worker_id"`
	RoleInVisit string    `db:"role_in_visit" json:"role_in_visit"`
	// Enrichment
	FirstName *string `db:"first_name" json:"first_name,omitempty"`
	LastName  *string `db:"last_name"  json:"last_name,omitempty"`
	RoleTitle *string `db:"role_title" json:"role_title,omitempty"`
}

type Filter struct {
	CompanyID *uuid.UUID
	PatientID *uuid.UUID
	Status    string
	DateFrom  *time.Time
	DateTo    *time.Time
}

type Repository interface {
	Create(ctx context.Context, v *Visit) error
	FindByID(ctx context.Context, id uuid.UUID) (*Visit, error)
	Update(ctx context.Context, v *Visit) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f Filter, limit, offset int) ([]*Visit, int64, error)
	HasPatientParticipation(ctx context.Context, visitID, patientID uuid.UUID) (bool, error)
	AssignWorkers(ctx context.Context, visitID uuid.UUID, workerIDs []uuid.UUID) error
	ListWorkers(ctx context.Context, visitID uuid.UUID) ([]*VisitWorker, error)
}
