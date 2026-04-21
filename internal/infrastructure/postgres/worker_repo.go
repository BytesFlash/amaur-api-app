package postgres

import (
	"context"
	"fmt"

	"amaur/api/internal/domain/worker"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type workerRepo struct {
	db *gorm.DB
}

func NewWorkerRepository(db *gorm.DB) worker.Repository {
	return &workerRepo{db: db}
}

func (r *workerRepo) Create(ctx context.Context, w *worker.Worker) error {
	values := map[string]interface{}{
		"id":                 w.ID,
		"user_id":            w.UserID,
		"rut":                w.RUT,
		"first_name":         w.FirstName,
		"last_name":          w.LastName,
		"email":              w.Email,
		"phone":              w.Phone,
		"role_title":         w.RoleTitle,
		"specialty":          w.Specialty,
		"hire_date":          w.HireDate,
		"birth_date":         w.BirthDate,
		"is_active":          w.IsActive,
		"availability_notes": w.AvailabilityNotes,
		"internal_notes":     w.InternalNotes,
		"created_by":         w.CreatedBy,
	}
	return r.db.WithContext(ctx).Table("amaur_workers").Create(values).Error
}

func (r *workerRepo) FindByID(ctx context.Context, id uuid.UUID) (*worker.Worker, error) {
	var w worker.Worker
	err := r.db.WithContext(ctx).
		Table("amaur_workers").
		Where("id = ? AND deleted_at IS NULL", id).
		Take(&w).Error
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *workerRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*worker.Worker, error) {
	var w worker.Worker
	err := r.db.WithContext(ctx).
		Table("amaur_workers").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Take(&w).Error
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *workerRepo) Update(ctx context.Context, w *worker.Worker) error {
	updates := map[string]interface{}{
		"rut":                w.RUT,
		"first_name":         w.FirstName,
		"last_name":          w.LastName,
		"email":              w.Email,
		"phone":              w.Phone,
		"role_title":         w.RoleTitle,
		"specialty":          w.Specialty,
		"hire_date":          w.HireDate,
		"birth_date":         w.BirthDate,
		"termination_date":   w.TerminationDate,
		"is_active":          w.IsActive,
		"availability_notes": w.AvailabilityNotes,
		"internal_notes":     w.InternalNotes,
		"updated_by":         w.UpdatedBy,
		"updated_at":         gorm.Expr("NOW()"),
	}
	return r.db.WithContext(ctx).
		Table("amaur_workers").
		Where("id = ? AND deleted_at IS NULL", w.ID).
		Updates(updates).Error
}

func (r *workerRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Table("amaur_workers").
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *workerRepo) ListActive(ctx context.Context, limit, offset int) ([]*worker.Worker, int64, error) {
	return r.List(ctx, "", "", true, limit, offset)
}

func (r *workerRepo) List(ctx context.Context, search string, specialtyCode string, onlyActive bool, limit, offset int) ([]*worker.Worker, int64, error) {
	query := r.db.WithContext(ctx).Table("amaur_workers AS w").Where("w.deleted_at IS NULL")
	if onlyActive {
		query = query.Where("w.is_active = ?", true)
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("(w.first_name ILIKE ? OR w.last_name ILIKE ? OR w.email ILIKE ?)", like, like, like)
	}
	if specialtyCode != "" {
		query = query.Where(
			"EXISTS (SELECT 1 FROM worker_specialties ws WHERE ws.worker_id = w.id AND ws.specialty_code = ?)",
			specialtyCode,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var workers []*worker.Worker
	if err := query.
		Select("w.*").
		Order("w.last_name ASC, w.first_name ASC").
		Limit(limit).
		Offset(offset).
		Scan(&workers).Error; err != nil {
		return nil, 0, err
	}

	if len(workers) > 0 {
		ids := make([]uuid.UUID, 0, len(workers))
		for _, w := range workers {
			ids = append(ids, w.ID)
		}
		specMap, err := r.bulkLoadSpecialties(ctx, ids)
		if err == nil {
			for _, w := range workers {
				if s, ok := specMap[w.ID]; ok {
					w.Specialties = s
				}
			}
		}
	}

	return workers, total, nil
}

func (r *workerRepo) LinkUser(ctx context.Context, workerID, userID uuid.UUID) error {
	return rawExec(ctx, r.db,
		`UPDATE amaur_workers SET user_id = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`,
		userID, workerID)
}

func (r *workerRepo) ListSpecialties(ctx context.Context) ([]worker.SpecialtyItem, error) {
	var items []worker.SpecialtyItem
	err := rawSelect(ctx, r.db, &items, `SELECT code, name FROM specialties WHERE is_active = true ORDER BY name`)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *workerRepo) CreateSpecialty(ctx context.Context, item worker.SpecialtyItem) error {
	return rawExec(ctx, r.db, `INSERT INTO specialties (code, name) VALUES ($1, $2)`, item.Code, item.Name)
}

func (r *workerRepo) DeleteSpecialty(ctx context.Context, code string) error {
	tx := r.db.WithContext(ctx).Exec(`DELETE FROM specialties WHERE code = $1`, code)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return fmt.Errorf("specialty not found")
	}
	return nil
}

func (r *workerRepo) GetWorkerSpecialties(ctx context.Context, workerID uuid.UUID) ([]worker.SpecialtyItem, error) {
	var items []worker.SpecialtyItem
	err := rawSelect(ctx, r.db, &items, `
		SELECT s.code, s.name
		FROM worker_specialties ws
		JOIN specialties s ON s.code = ws.specialty_code
		WHERE ws.worker_id = $1
		ORDER BY s.name`, workerID)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *workerRepo) SetWorkerSpecialties(ctx context.Context, workerID uuid.UUID, codes []string, setBy uuid.UUID) error {
	return withTx(ctx, r.db, func(tx *gorm.DB) error {
		if err := rawExec(ctx, tx, `DELETE FROM worker_specialties WHERE worker_id = $1`, workerID); err != nil {
			return err
		}
		for _, code := range codes {
			if err := rawExec(ctx, tx, `
				INSERT INTO worker_specialties (id, worker_id, specialty_code, created_by)
				VALUES (uuid_generate_v4(), $1, $2, $3)
				ON CONFLICT (worker_id, specialty_code) DO NOTHING`,
				workerID, code, setBy); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *workerRepo) bulkLoadSpecialties(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]worker.SpecialtyItem, error) {
	type row struct {
		WorkerID uuid.UUID `gorm:"column:worker_id"`
		Code     string    `gorm:"column:code"`
		Name     string    `gorm:"column:name"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).Raw(`
		SELECT ws.worker_id, s.code, s.name
		FROM worker_specialties ws
		JOIN specialties s ON s.code = ws.specialty_code
		WHERE ws.worker_id IN ?
		ORDER BY ws.worker_id, s.name`, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID][]worker.SpecialtyItem)
	for _, row := range rows {
		result[row.WorkerID] = append(result[row.WorkerID], worker.SpecialtyItem{
			Code: row.Code,
			Name: row.Name,
		})
	}
	return result, nil
}

func (r *workerRepo) ListAvailabilityRules(ctx context.Context, workerID uuid.UUID) ([]*worker.AvailabilityRule, error) {
	rules := make([]*worker.AvailabilityRule, 0)
	err := rawSelectPtr(ctx, r.db, &rules, `
		SELECT id, worker_id, weekday,
		       TO_CHAR(start_time, 'HH24:MI') AS start_time,
		       TO_CHAR(end_time,   'HH24:MI') AS end_time,
		       is_active, created_at, created_by
		FROM worker_availability_rules
		WHERE worker_id = $1 AND is_active = TRUE
		ORDER BY weekday, start_time`, workerID)
	return rules, err
}

func (r *workerRepo) ReplaceAvailabilityRules(ctx context.Context, workerID uuid.UUID, rules []*worker.AvailabilityRule) error {
	return withTx(ctx, r.db, func(tx *gorm.DB) error {
		if err := rawExec(ctx, tx, `DELETE FROM worker_availability_rules WHERE worker_id = $1`, workerID); err != nil {
			return err
		}
		for _, rule := range rules {
			if err := rawExec(ctx, tx, `
				INSERT INTO worker_availability_rules
					(id, worker_id, weekday, start_time, end_time, is_active, created_by)
				VALUES ($1, $2, $3, $4::TIME, $5::TIME, TRUE, $6)`,
				rule.ID, workerID, rule.Weekday, rule.StartTime, rule.EndTime, rule.CreatedBy); err != nil {
				return err
			}
		}
		return nil
	})
}
