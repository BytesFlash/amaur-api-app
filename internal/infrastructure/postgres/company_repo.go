package postgres

import (
	"context"
	"fmt"
	"strings"

	"amaur/api/internal/domain/company"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type companyRepo struct {
	db *sqlx.DB
}

func NewCompanyRepository(db *sqlx.DB) company.Repository {
	return &companyRepo{db: db}
}

func (r *companyRepo) Create(ctx context.Context, c *company.Company) error {
	query := `
		INSERT INTO companies (
			id, rut, name, fantasy_name, industry, size_category,
			contact_name, contact_email, contact_phone, billing_email,
			address, city, region, website, status, commercial_notes,
			lead_source, created_by
		) VALUES (
			:id, :rut, :name, :fantasy_name, :industry, :size_category,
			:contact_name, :contact_email, :contact_phone, :billing_email,
			:address, :city, :region, :website, :status, :commercial_notes,
			:lead_source, :created_by
		)`
	_, err := r.db.NamedExecContext(ctx, query, c)
	return err
}

func (r *companyRepo) FindByID(ctx context.Context, id uuid.UUID) (*company.Company, error) {
	var c company.Company
	err := r.db.GetContext(ctx, &c,
		`SELECT * FROM companies WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *companyRepo) FindByRUT(ctx context.Context, rut string) (*company.Company, error) {
	var c company.Company
	err := r.db.GetContext(ctx, &c,
		`SELECT * FROM companies WHERE rut = $1 AND deleted_at IS NULL`, rut)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *companyRepo) Update(ctx context.Context, c *company.Company) error {
	query := `
		UPDATE companies SET
			rut = :rut,
			name = :name,
			fantasy_name = :fantasy_name,
			industry = :industry,
			size_category = :size_category,
			contact_name = :contact_name,
			contact_email = :contact_email,
			contact_phone = :contact_phone,
			billing_email = :billing_email,
			address = :address,
			city = :city,
			region = :region,
			website = :website,
			status = :status,
			commercial_notes = :commercial_notes,
			lead_source = :lead_source,
			updated_by = :updated_by,
			updated_at = NOW()
		WHERE id = :id AND deleted_at IS NULL`
	_, err := r.db.NamedExecContext(ctx, query, c)
	return err
}

func (r *companyRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE companies SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}

func (r *companyRepo) List(ctx context.Context, f company.Filter, limit, offset int) ([]*company.Company, int64, error) {
	args := []interface{}{}
	where := []string{"deleted_at IS NULL"}
	idx := 1

	if f.Search != "" {
		where = append(where, fmt.Sprintf(
			`(name ILIKE $%d OR fantasy_name ILIKE $%d OR rut ILIKE $%d)`, idx, idx+1, idx+2))
		like := "%" + f.Search + "%"
		args = append(args, like, like, like)
		idx += 3
	}
	if f.Status != "" {
		where = append(where, fmt.Sprintf(`status = $%d`, idx))
		args = append(args, f.Status)
		idx++
	}
	if f.Region != "" {
		where = append(where, fmt.Sprintf(`region = $%d`, idx))
		args = append(args, f.Region)
		idx++
	}
	if f.Industry != "" {
		where = append(where, fmt.Sprintf(`industry ILIKE $%d`, idx))
		args = append(args, "%"+f.Industry+"%")
		idx++
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM companies `+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows, err := r.db.QueryxContext(ctx,
		`SELECT * FROM companies `+whereClause+
			fmt.Sprintf(` ORDER BY name ASC LIMIT $%d OFFSET $%d`, idx, idx+1),
		args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var companies []*company.Company
	for rows.Next() {
		var c company.Company
		if err := rows.StructScan(&c); err != nil {
			return nil, 0, err
		}
		companies = append(companies, &c)
	}
	return companies, total, rows.Err()
}

func (r *companyRepo) CreateBranch(ctx context.Context, b *company.Branch) error {
	query := `
		INSERT INTO company_branches (
			id, company_id, name, address, city, region,
			contact_name, contact_phone, is_main, is_active
		) VALUES (
			:id, :company_id, :name, :address, :city, :region,
			:contact_name, :contact_phone, :is_main, :is_active
		)`
	_, err := r.db.NamedExecContext(ctx, query, b)
	return err
}

func (r *companyRepo) UpdateBranch(ctx context.Context, b *company.Branch) error {
	query := `
		UPDATE company_branches SET
			name = :name,
			address = :address,
			city = :city,
			region = :region,
			contact_name = :contact_name,
			contact_phone = :contact_phone,
			is_main = :is_main,
			is_active = :is_active
		WHERE id = :id`
	_, err := r.db.NamedExecContext(ctx, query, b)
	return err
}

func (r *companyRepo) ListBranches(ctx context.Context, companyID uuid.UUID) ([]*company.Branch, error) {
	var branches []*company.Branch
	err := r.db.SelectContext(ctx, &branches,
		`SELECT * FROM company_branches WHERE company_id = $1 ORDER BY is_main DESC, name ASC`,
		companyID)
	return branches, err
}

// ExistsByIDs returns the subset of ids that do NOT correspond to an active
// company in the database. An empty return slice means all ids are valid.
func (r *companyRepo) ExistsByIDs(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query := `SELECT id FROM companies WHERE id IN (` +
		strings.Join(placeholders, ",") +
		`) AND deleted_at IS NULL`

	var found []uuid.UUID
	if err := r.db.SelectContext(ctx, &found, query, args...); err != nil {
		return nil, err
	}

	foundSet := make(map[uuid.UUID]struct{}, len(found))
	for _, id := range found {
		foundSet[id] = struct{}{}
	}
	var missing []uuid.UUID
	for _, id := range ids {
		if _, ok := foundSet[id]; !ok {
			missing = append(missing, id)
		}
	}
	return missing, nil
}

func (r *companyRepo) ListPatients(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]*company.PatientSummary, int64, error) {
	var total int64
	if err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM patient_companies pc
		 JOIN patients p ON p.id = pc.patient_id
		 WHERE pc.company_id = $1 AND pc.is_active = true AND p.deleted_at IS NULL`,
		companyID); err != nil {
		return nil, 0, err
	}
	var rows []*company.PatientSummary
	err := r.db.SelectContext(ctx, &rows, `
		SELECT p.id, p.first_name, p.last_name, p.rut, p.email, p.phone,
			p.status, p.patient_type, pc.position, pc.department
		FROM patient_companies pc
		JOIN patients p ON p.id = pc.patient_id
		WHERE pc.company_id = $1 AND pc.is_active = true AND p.deleted_at IS NULL
		ORDER BY p.first_name, p.last_name
		LIMIT $2 OFFSET $3`, companyID, limit, offset)
	return rows, total, err
}
