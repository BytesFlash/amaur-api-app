package postgres

import (
	"context"
	"fmt"
	"strings"

	"amaur/api/internal/domain/company"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type companyRepo struct {
	db *gorm.DB
}

func NewCompanyRepository(db *gorm.DB) company.Repository {
	return &companyRepo{db: db}
}

func (r *companyRepo) Create(ctx context.Context, c *company.Company) error {
	return rawExec(ctx, r.db, `
		INSERT INTO companies (
			id, rut, name, fantasy_name, industry, size_category,
			contact_name, contact_email, contact_phone, billing_email,
			address, city, region, website, status, commercial_notes,
			lead_source, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16,
			$17, $18
		)`,
		c.ID, c.RUT, c.Name, c.FantasyName, c.Industry, c.SizeCategory,
		c.ContactName, c.ContactEmail, c.ContactPhone, c.BillingEmail,
		c.Address, c.City, c.Region, c.Website, c.Status, c.CommercialNotes,
		c.LeadSource, c.CreatedBy)
}

func (r *companyRepo) FindByID(ctx context.Context, id uuid.UUID) (*company.Company, error) {
	var c company.Company
	err := rawGet(ctx, r.db, &c, `SELECT * FROM companies WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *companyRepo) FindByRUT(ctx context.Context, rut string) (*company.Company, error) {
	var c company.Company
	err := rawGet(ctx, r.db, &c, `SELECT * FROM companies WHERE rut = $1 AND deleted_at IS NULL`, rut)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *companyRepo) Update(ctx context.Context, c *company.Company) error {
	return rawExec(ctx, r.db, `
		UPDATE companies SET
			rut = $1,
			name = $2,
			fantasy_name = $3,
			industry = $4,
			size_category = $5,
			contact_name = $6,
			contact_email = $7,
			contact_phone = $8,
			billing_email = $9,
			address = $10,
			city = $11,
			region = $12,
			website = $13,
			status = $14,
			commercial_notes = $15,
			lead_source = $16,
			updated_by = $17,
			updated_at = NOW()
		WHERE id = $18 AND deleted_at IS NULL`,
		c.RUT, c.Name, c.FantasyName, c.Industry, c.SizeCategory,
		c.ContactName, c.ContactEmail, c.ContactPhone, c.BillingEmail,
		c.Address, c.City, c.Region, c.Website, c.Status, c.CommercialNotes,
		c.LeadSource, c.UpdatedBy, c.ID)
}

func (r *companyRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return rawExec(ctx, r.db, `UPDATE companies SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
}

func (r *companyRepo) List(ctx context.Context, f company.Filter, limit, offset int) ([]*company.Company, int64, error) {
	args := []interface{}{}
	where := []string{"deleted_at IS NULL"}
	idx := 1

	if f.ID != nil {
		where = append(where, fmt.Sprintf(`id = $%d`, idx))
		args = append(args, *f.ID)
		idx++
	}
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

	var totalRow struct {
		Count int64 `gorm:"column:count"`
	}
	if err := rawGet(ctx, r.db, &totalRow, `SELECT COUNT(*) AS count FROM companies `+whereClause, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	var companies []*company.Company
	if err := rawSelectPtr(ctx, r.db, &companies,
		`SELECT * FROM companies `+whereClause+
			fmt.Sprintf(` ORDER BY name ASC LIMIT $%d OFFSET $%d`, idx, idx+1),
		args...); err != nil {
		return nil, 0, err
	}
	return companies, totalRow.Count, nil
}

func (r *companyRepo) CreateBranch(ctx context.Context, b *company.Branch) error {
	return rawExec(ctx, r.db, `
		INSERT INTO company_branches (
			id, company_id, name, address, city, region,
			contact_name, contact_phone, is_main, is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10
		)`,
		b.ID, b.CompanyID, b.Name, b.Address, b.City, b.Region,
		b.ContactName, b.ContactPhone, b.IsMain, b.IsActive)
}

func (r *companyRepo) UpdateBranch(ctx context.Context, b *company.Branch) error {
	return rawExec(ctx, r.db, `
		UPDATE company_branches SET
			name = $1,
			address = $2,
			city = $3,
			region = $4,
			contact_name = $5,
			contact_phone = $6,
			is_main = $7,
			is_active = $8
		WHERE id = $9`,
		b.Name, b.Address, b.City, b.Region, b.ContactName, b.ContactPhone, b.IsMain, b.IsActive, b.ID)
}

func (r *companyRepo) ListBranches(ctx context.Context, companyID uuid.UUID) ([]*company.Branch, error) {
	var branches []*company.Branch
	err := rawSelectPtr(ctx, r.db, &branches,
		`SELECT * FROM company_branches WHERE company_id = $1 ORDER BY is_main DESC, name ASC`,
		companyID)
	return branches, err
}

func (r *companyRepo) ExistsByIDs(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	type row struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Raw(`SELECT id FROM companies WHERE id IN ? AND deleted_at IS NULL`, ids).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	foundSet := make(map[uuid.UUID]struct{}, len(rows))
	for _, row := range rows {
		foundSet[row.ID] = struct{}{}
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
	var totalRow struct {
		Count int64 `gorm:"column:count"`
	}
	if err := rawGet(ctx, r.db, &totalRow,
		`SELECT COUNT(*) AS count FROM patient_companies pc
		 JOIN patients p ON p.id = pc.patient_id
		 WHERE pc.company_id = $1 AND pc.is_active = true AND p.deleted_at IS NULL`,
		companyID); err != nil {
		return nil, 0, err
	}

	var rows []*company.PatientSummary
	err := rawSelectPtr(ctx, r.db, &rows, `
		SELECT p.id, p.first_name, p.last_name, p.rut, p.email, p.phone,
			p.status, p.patient_type, pc.position, pc.department
		FROM patient_companies pc
		JOIN patients p ON p.id = pc.patient_id
		WHERE pc.company_id = $1 AND pc.is_active = true AND p.deleted_at IS NULL
		ORDER BY p.first_name, p.last_name
		LIMIT $2 OFFSET $3`, companyID, limit, offset)
	return rows, totalRow.Count, err
}
