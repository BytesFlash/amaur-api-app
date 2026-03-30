package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"amaur/api/internal/domain/contract"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type contractRepo struct {
	db *sqlx.DB
}

func NewContractRepository(db *sqlx.DB) contract.Repository {
	return &contractRepo{db: db}
}

func (r *contractRepo) Create(ctx context.Context, c *contract.Contract) error {
	c.ID = uuid.New()
	now := time.Now()
	c.CreatedAt = now
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO contracts (id, company_id, name, contract_type, status,
			start_date, end_date, renewal_date, value_clp, billing_cycle,
			notes, signed_document_url, created_at, created_by)
		VALUES (:id, :company_id, :name, :contract_type, :status,
			:start_date, :end_date, :renewal_date, :value_clp, :billing_cycle,
			:notes, :signed_document_url, :created_at, :created_by)
	`, c)
	return err
}

func (r *contractRepo) FindByID(ctx context.Context, id uuid.UUID) (*contract.Contract, error) {
	c := &contract.Contract{}
	err := r.db.GetContext(ctx, c, `SELECT * FROM contracts WHERE id=$1`, id)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *contractRepo) Update(ctx context.Context, c *contract.Contract) error {
	now := time.Now()
	c.UpdatedAt = &now
	_, err := r.db.NamedExecContext(ctx, `
		UPDATE contracts SET
			name=:name, contract_type=:contract_type, status=:status,
			start_date=:start_date, end_date=:end_date, renewal_date=:renewal_date,
			value_clp=:value_clp, billing_cycle=:billing_cycle, notes=:notes,
			signed_document_url=:signed_document_url,
			updated_at=:updated_at, updated_by=:updated_by
		WHERE id=:id
	`, c)
	return err
}

func (r *contractRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM contracts WHERE id=$1`, id)
	return err
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

	var total int64
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM contracts WHERE `+clause, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows := []*contract.Contract{}
	if err := r.db.SelectContext(ctx, &rows,
		fmt.Sprintf(`SELECT * FROM contracts WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, clause, idx, idx+1),
		args...); err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *contractRepo) ListServices(ctx context.Context, contractID uuid.UUID) ([]*contract.ContractService, error) {
	rows := []*contract.ContractService{}
	err := r.db.SelectContext(ctx, &rows, `SELECT * FROM contract_services WHERE contract_id=$1`, contractID)
	return rows, err
}
