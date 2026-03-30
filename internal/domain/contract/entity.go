package contract

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Contract struct {
	ID                uuid.UUID  `db:"id"                  json:"id"`
	CompanyID         uuid.UUID  `db:"company_id"          json:"company_id"`
	Name              string     `db:"name"                json:"name"`
	ContractType      *string    `db:"contract_type"       json:"contract_type,omitempty"`
	Status            string     `db:"status"              json:"status"`
	StartDate         time.Time  `db:"start_date"          json:"start_date"`
	EndDate           *time.Time `db:"end_date"            json:"end_date,omitempty"`
	RenewalDate       *time.Time `db:"renewal_date"        json:"renewal_date,omitempty"`
	ValueCLP          *float64   `db:"value_clp"           json:"value_clp,omitempty"`
	BillingCycle      *string    `db:"billing_cycle"       json:"billing_cycle,omitempty"`
	Notes             *string    `db:"notes"               json:"notes,omitempty"`
	SignedDocumentURL *string    `db:"signed_document_url" json:"signed_document_url,omitempty"`
	CreatedAt         time.Time  `db:"created_at"          json:"created_at"`
	UpdatedAt         *time.Time `db:"updated_at"          json:"updated_at,omitempty"`
	CreatedBy         *uuid.UUID `db:"created_by"          json:"-"`
	UpdatedBy         *uuid.UUID `db:"updated_by"          json:"-"`
}

type ContractService struct {
	ID                uuid.UUID `db:"id"                   json:"id"`
	ContractID        uuid.UUID `db:"contract_id"          json:"contract_id"`
	ServiceTypeID     uuid.UUID `db:"service_type_id"      json:"service_type_id"`
	QuotaType         string    `db:"quota_type"           json:"quota_type"`
	QuantityPerPeriod *int      `db:"quantity_per_period"  json:"quantity_per_period,omitempty"`
	PeriodUnit        *string   `db:"period_unit"          json:"period_unit,omitempty"`
	SessionsIncluded  *int      `db:"sessions_included"    json:"sessions_included,omitempty"`
	SessionsUsed      int       `db:"sessions_used"        json:"sessions_used"`
	HoursIncluded     *float64  `db:"hours_included"       json:"hours_included,omitempty"`
	HoursUsed         float64   `db:"hours_used"           json:"hours_used"`
	PricePerUnit      *float64  `db:"price_per_unit"       json:"price_per_unit,omitempty"`
	Notes             *string   `db:"notes"                json:"notes,omitempty"`
}

type Filter struct {
	CompanyID *uuid.UUID
	Status    string
}

type Repository interface {
	Create(ctx context.Context, c *Contract) error
	FindByID(ctx context.Context, id uuid.UUID) (*Contract, error)
	Update(ctx context.Context, c *Contract) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f Filter, limit, offset int) ([]*Contract, int64, error)
	ListServices(ctx context.Context, contractID uuid.UUID) ([]*ContractService, error)
}
