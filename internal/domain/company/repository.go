package company

import (
	"context"

	"github.com/google/uuid"
)

type Filter struct {
	Search   string
	Status   string
	Region   string
	Industry string
}

type Repository interface {
	Create(ctx context.Context, c *Company) error
	FindByID(ctx context.Context, id uuid.UUID) (*Company, error)
	FindByRUT(ctx context.Context, rut string) (*Company, error)
	Update(ctx context.Context, c *Company) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f Filter, limit, offset int) ([]*Company, int64, error)

	CreateBranch(ctx context.Context, b *Branch) error
	UpdateBranch(ctx context.Context, b *Branch) error
	ListBranches(ctx context.Context, companyID uuid.UUID) ([]*Branch, error)
	ListPatients(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]*PatientSummary, int64, error)

	// ExistsByIDs returns the subset of ids that do NOT exist as active companies.
	// An empty slice means all ids are valid.
	ExistsByIDs(ctx context.Context, ids []uuid.UUID) (missing []uuid.UUID, err error)
}
