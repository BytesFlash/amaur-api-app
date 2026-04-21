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
	err := r.db.WithContext(ctx).
		Table("contract_services AS cs").
		Select(`
			cs.id,
			cs.contract_id,
			cs.service_type_id,
			cs.quota_type,
			cs.quantity_per_period,
			cs.period_unit,
			cs.sessions_included,
			cs.sessions_used,
			cs.hours_included,
			cs.hours_used,
			cs.price_per_unit,
			cs.notes,
			st.name AS service_type_name`).
		Joins("JOIN service_types st ON st.id = cs.service_type_id").
		Where("cs.contract_id = ?", contractID).
		Order("st.name ASC, cs.id ASC").
		Scan(&rows).Error
	return rows, err
}

func (r *contractRepo) UpsertServices(ctx context.Context, contractID uuid.UUID, services []*contract.ContractService) error {
	return withTx(ctx, r.db, func(tx *gorm.DB) error {
		var existing []*contract.ContractService
		if err := tx.WithContext(ctx).
			Table("contract_services").
			Where("contract_id = ?", contractID).
			Find(&existing).Error; err != nil {
			return err
		}

		existingByID := make(map[uuid.UUID]*contract.ContractService, len(existing))
		for _, item := range existing {
			existingByID[item.ID] = item
		}

		keepIDs := make([]uuid.UUID, 0, len(services))
		for _, svc := range services {
			if svc.ID != uuid.Nil {
				keepIDs = append(keepIDs, svc.ID)
			}
		}

		deleteQuery := tx.WithContext(ctx).Table("contract_services").Where("contract_id = ?", contractID)
		if len(keepIDs) > 0 {
			deleteQuery = deleteQuery.Where("id NOT IN ?", keepIDs)
		}
		if err := deleteQuery.Delete(&contract.ContractService{}).Error; err != nil {
			return err
		}

		for _, svc := range services {
			quotaType := svc.QuotaType
			if quotaType == "" {
				quotaType = "sessions"
			}

			values := map[string]interface{}{
				"contract_id":         contractID,
				"service_type_id":     svc.ServiceTypeID,
				"quota_type":          quotaType,
				"quantity_per_period": svc.QuantityPerPeriod,
				"period_unit":         svc.PeriodUnit,
				"sessions_included":   svc.SessionsIncluded,
				"hours_included":      svc.HoursIncluded,
				"price_per_unit":      svc.PricePerUnit,
				"notes":               svc.Notes,
			}

			if svc.ID != uuid.Nil {
				if _, ok := existingByID[svc.ID]; ok {
					if err := tx.WithContext(ctx).
						Table("contract_services").
						Where("id = ? AND contract_id = ?", svc.ID, contractID).
						Updates(values).Error; err != nil {
						return err
					}
					continue
				}
			}

			newID := svc.ID
			if newID == uuid.Nil {
				newID = uuid.New()
			}
			values["id"] = newID
			if err := tx.WithContext(ctx).Table("contract_services").Create(values).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
