package postgres

import (
	"context"
	"time"

	"amaur/api/internal/domain/servicetype"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type serviceTypeRepo struct {
	db *sqlx.DB
}

func NewServiceTypeRepository(db *sqlx.DB) servicetype.Repository {
	return &serviceTypeRepo{db: db}
}

func (r *serviceTypeRepo) List(ctx context.Context, activeOnly bool) ([]*servicetype.ServiceType, error) {
	query := `SELECT id, name, category, description, default_duration_minutes,
		is_group_service, requires_clinical_record, is_active, created_at, updated_at
		FROM service_types`
	if activeOnly {
		query += ` WHERE is_active = true`
	}
	query += ` ORDER BY category, name`
	var rows []*servicetype.ServiceType
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, err
	}

	// Bulk-load specialties to avoid N+1.
	if len(rows) > 0 {
		ids := make([]uuid.UUID, len(rows))
		for i, st := range rows {
			ids[i] = st.ID
		}
		specMap, err := r.bulkLoadSpecialties(ctx, ids)
		if err == nil {
			for _, st := range rows {
				if s, ok := specMap[st.ID]; ok {
					st.Specialties = s
				}
			}
		}
	}
	return rows, nil
}

func (r *serviceTypeRepo) FindByID(ctx context.Context, id uuid.UUID) (*servicetype.ServiceType, error) {
	var st servicetype.ServiceType
	err := r.db.GetContext(ctx, &st, `
		SELECT id, name, category, description, default_duration_minutes,
			is_group_service, requires_clinical_record, is_active, created_at, updated_at
		FROM service_types WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	specs, err := r.getSpecialties(ctx, id)
	if err == nil {
		st.Specialties = specs
	}
	return &st, nil
}

func (r *serviceTypeRepo) Create(ctx context.Context, st *servicetype.ServiceType) error {
	st.ID = uuid.New()
	st.CreatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO service_types (id, name, category, description, default_duration_minutes,
			is_group_service, requires_clinical_record, is_active, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		st.ID, st.Name, st.Category, st.Description, st.DefaultDurationMinutes,
		st.IsGroupService, st.RequiresClinicalRecord, st.IsActive, st.CreatedAt)
	return err
}

func (r *serviceTypeRepo) Update(ctx context.Context, st *servicetype.ServiceType) error {
	now := time.Now()
	st.UpdatedAt = &now
	_, err := r.db.ExecContext(ctx, `
		UPDATE service_types SET name=$1, category=$2, description=$3,
			default_duration_minutes=$4, is_group_service=$5,
			requires_clinical_record=$6, is_active=$7, updated_at=$8
		WHERE id=$9`,
		st.Name, st.Category, st.Description, st.DefaultDurationMinutes,
		st.IsGroupService, st.RequiresClinicalRecord, st.IsActive, st.UpdatedAt, st.ID)
	return err
}

// SetSpecialties atomically replaces all specialties for a service type.
func (r *serviceTypeRepo) SetSpecialties(ctx context.Context, serviceTypeID uuid.UUID, codes []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM service_type_specialties WHERE service_type_id = $1`, serviceTypeID); err != nil {
		return err
	}
	for _, code := range codes {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO service_type_specialties (id, service_type_id, specialty_code)
			VALUES (uuid_generate_v4(), $1, $2)
			ON CONFLICT (service_type_id, specialty_code) DO NOTHING`,
			serviceTypeID, code); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// getSpecialties returns specialties linked to the given service type.
func (r *serviceTypeRepo) getSpecialties(ctx context.Context, id uuid.UUID) ([]servicetype.SpecialtyItem, error) {
	var items []servicetype.SpecialtyItem
	err := r.db.SelectContext(ctx, &items, `
		SELECT s.code, s.name
		FROM service_type_specialties sts
		JOIN specialties s ON s.code = sts.specialty_code
		WHERE sts.service_type_id = $1
		ORDER BY s.name`, id)
	return items, err
}

// bulkLoadSpecialties fetches specialties for a set of service type IDs in one query.
func (r *serviceTypeRepo) bulkLoadSpecialties(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]servicetype.SpecialtyItem, error) {
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = id.String()
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT sts.service_type_id, s.code, s.name
		FROM service_type_specialties sts
		JOIN specialties s ON s.code = sts.specialty_code
		WHERE sts.service_type_id = ANY($1::uuid[])
		ORDER BY sts.service_type_id, s.name`,
		pq.Array(strs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]servicetype.SpecialtyItem)
	for rows.Next() {
		var stIDStr string
		var item servicetype.SpecialtyItem
		if err := rows.Scan(&stIDStr, &item.Code, &item.Name); err != nil {
			return nil, err
		}
		stID, _ := uuid.Parse(stIDStr)
		result[stID] = append(result[stID], item)
	}
	return result, rows.Err()
}
