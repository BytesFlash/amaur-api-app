package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"amaur/api/internal/domain/contract"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type contractRepo struct {
	db *gorm.DB
}

func NewContractRepository(db *gorm.DB) contract.Repository {
	return &contractRepo{db: db}
}

func (r *contractRepo) Create(ctx context.Context, c *contract.Contract) error {
	c.ID = uuid.New()
	now := time.Now()
	c.CreatedAt = now
	return rawExec(ctx, r.db, `
		INSERT INTO contracts (id, company_id, name, contract_type, status,
			start_date, end_date, renewal_date, value_clp, billing_cycle,
			notes, signed_document_url, created_at, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	`, c.ID, c.CompanyID, c.Name, c.ContractType, c.Status,
		c.StartDate, c.EndDate, c.RenewalDate, c.ValueCLP, c.BillingCycle,
		c.Notes, c.SignedDocumentURL, c.CreatedAt, c.CreatedBy)
}

func (r *contractRepo) FindByID(ctx context.Context, id uuid.UUID) (*contract.Contract, error) {
	c := &contract.Contract{}
	err := rawGet(ctx, r.db, c, `SELECT * FROM contracts WHERE id=$1`, id)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *contractRepo) Update(ctx context.Context, c *contract.Contract) error {
	now := time.Now()
	c.UpdatedAt = &now
	return rawExec(ctx, r.db, `
		UPDATE contracts SET
			name=$1, contract_type=$2, status=$3,
			start_date=$4, end_date=$5, renewal_date=$6,
			value_clp=$7, billing_cycle=$8, notes=$9,
			signed_document_url=$10,
			updated_at=$11, updated_by=$12
		WHERE id=$13
	`, c.Name, c.ContractType, c.Status,
		c.StartDate, c.EndDate, c.RenewalDate,
		c.ValueCLP, c.BillingCycle, c.Notes,
		c.SignedDocumentURL, c.UpdatedAt, c.UpdatedBy, c.ID)
}

func (r *contractRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return rawExec(ctx, r.db, `DELETE FROM contracts WHERE id=$1`, id)
}

func (r *contractRepo) List(ctx context.Context, f contract.Filter, limit, offset int) ([]*contract.Contract, int64, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if f.CompanyID != nil {
		where = append(where, fmt.Sprintf("company_id=$%d", idx))
		args = append(args, *f.CompanyID)
		idx++
	}
	if f.Status != "" {
		where = append(where, fmt.Sprintf("status=$%d", idx))
		args = append(args, f.Status)
		idx++
	}

	clause := strings.Join(where, " AND ")

	var totalRow struct {
		Count int64 `gorm:"column:count"`
	}
	if err := rawGet(ctx, r.db, &totalRow, `SELECT COUNT(*) AS count FROM contracts WHERE `+clause, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows := []*contract.Contract{}
	if err := rawSelectPtr(ctx, r.db, &rows,
		fmt.Sprintf(`SELECT * FROM contracts WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, clause, idx, idx+1),
		args...); err != nil {
		return nil, 0, err
	}

	return rows, totalRow.Count, nil
}

func (r *contractRepo) ListServices(ctx context.Context, contractID uuid.UUID) ([]*contract.ContractService, error) {
	rows := []*contract.ContractService{}
	err := rawSelectPtr(ctx, r.db, &rows, `SELECT * FROM contract_services WHERE contract_id=$1`, contractID)
	return rows, err
}
