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
	return rawExec(ctx, r.db, `
		INSERT INTO amaur_workers (
			id, user_id, rut, first_name, last_name, email, phone,
			role_title, specialty, hire_date, birth_date, is_active,
			availability_notes, internal_notes, created_by
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,
			$8,$9,$10,$11,$12,
			$13,$14,$15
		)`,
		w.ID, w.UserID, w.RUT, w.FirstName, w.LastName, w.Email, w.Phone,
		w.RoleTitle, w.Specialty, w.HireDate, w.BirthDate, w.IsActive,
		w.AvailabilityNotes, w.InternalNotes, w.CreatedBy)
}

func (r *workerRepo) FindByID(ctx context.Context, id uuid.UUID) (*worker.Worker, error) {
	var w worker.Worker
	err := rawGet(ctx, r.db, &w, `SELECT * FROM amaur_workers WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *workerRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*worker.Worker, error) {
	var w worker.Worker
	err := rawGet(ctx, r.db, &w, `SELECT * FROM amaur_workers WHERE user_id = $1 AND deleted_at IS NULL`, userID)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *workerRepo) Update(ctx context.Context, w *worker.Worker) error {
	return rawExec(ctx, r.db, `
		UPDATE amaur_workers SET
			rut = $1,
			first_name = $2,
			last_name = $3,
			email = $4,
			phone = $5,
			role_title = $6,
			specialty = $7,
			hire_date = $8,
			birth_date = $9,
			termination_date = $10,
			is_active = $11,
			availability_notes = $12,
			internal_notes = $13,
			updated_by = $14,
			updated_at = NOW()
		WHERE id = $15 AND deleted_at IS NULL`,
		w.RUT, w.FirstName, w.LastName, w.Email, w.Phone,
		w.RoleTitle, w.Specialty, w.HireDate, w.BirthDate, w.TerminationDate,
		w.IsActive, w.AvailabilityNotes, w.InternalNotes, w.UpdatedBy, w.ID)
}

func (r *workerRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return rawExec(ctx, r.db, `UPDATE amaur_workers SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
}

func (r *workerRepo) ListActive(ctx context.Context, limit, offset int) ([]*worker.Worker, int64, error) {
	return r.List(ctx, "", "", true, limit, offset)
}

func (r *workerRepo) List(ctx context.Context, search string, specialtyCode string, onlyActive bool, limit, offset int) ([]*worker.Worker, int64, error) {
	args := []interface{}{}
	where := "w.deleted_at IS NULL"
	idx := 1

	if onlyActive {
		where += " AND w.is_active = TRUE"
	}
	if search != "" {
		where += fmt.Sprintf(
			` AND (w.first_name ILIKE $%d OR w.last_name ILIKE $%d OR w.email ILIKE $%d)`,
			idx, idx+1, idx+2)
		like := "%" + search + "%"
		args = append(args, like, like, like)
		idx += 3
	}
	if specialtyCode != "" {
		where += fmt.Sprintf(
			` AND EXISTS (SELECT 1 FROM worker_specialties ws WHERE ws.worker_id = w.id AND ws.specialty_code = $%d)`,
			idx)
		args = append(args, specialtyCode)
		idx++
	}

	var totalRow struct {
		Count int64 `gorm:"column:count"`
	}
	if err := rawGet(ctx, r.db, &totalRow, `SELECT COUNT(*) AS count FROM amaur_workers w WHERE `+where, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	var workers []*worker.Worker
	if err := rawSelectPtr(ctx, r.db, &workers,
		`SELECT w.* FROM amaur_workers w WHERE `+where+
			fmt.Sprintf(` ORDER BY w.last_name ASC, w.first_name ASC LIMIT $%d OFFSET $%d`, idx, idx+1),
		args...); err != nil {
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

	return workers, totalRow.Count, nil
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
