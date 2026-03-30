package postgres

import (
	"context"
	"fmt"

	"amaur/api/internal/domain/worker"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type workerRepo struct {
	db *sqlx.DB
}

func NewWorkerRepository(db *sqlx.DB) worker.Repository {
	return &workerRepo{db: db}
}

func (r *workerRepo) Create(ctx context.Context, w *worker.Worker) error {
	query := `
		INSERT INTO amaur_workers (
			id, user_id, rut, first_name, last_name, email, phone,
			role_title, specialty, hire_date, birth_date, is_active,
			availability_notes, internal_notes, created_by
		) VALUES (
			:id, :user_id, :rut, :first_name, :last_name, :email, :phone,
			:role_title, :specialty, :hire_date, :birth_date, :is_active,
			:availability_notes, :internal_notes, :created_by
		)`
	_, err := r.db.NamedExecContext(ctx, query, w)
	return err
}

func (r *workerRepo) FindByID(ctx context.Context, id uuid.UUID) (*worker.Worker, error) {
	var w worker.Worker
	err := r.db.GetContext(ctx, &w,
		`SELECT * FROM amaur_workers WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *workerRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*worker.Worker, error) {
	var w worker.Worker
	err := r.db.GetContext(ctx, &w,
		`SELECT * FROM amaur_workers WHERE user_id = $1 AND deleted_at IS NULL`, userID)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *workerRepo) Update(ctx context.Context, w *worker.Worker) error {
	query := `
		UPDATE amaur_workers SET
			rut = :rut,
			first_name = :first_name,
			last_name = :last_name,
			email = :email,
			phone = :phone,
			role_title = :role_title,
			specialty = :specialty,
			hire_date = :hire_date,
			birth_date = :birth_date,
			termination_date = :termination_date,
			is_active = :is_active,
			availability_notes = :availability_notes,
			internal_notes = :internal_notes,
			updated_by = :updated_by,
			updated_at = NOW()
		WHERE id = :id AND deleted_at IS NULL`
	_, err := r.db.NamedExecContext(ctx, query, w)
	return err
}

func (r *workerRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE amaur_workers SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
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

	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM amaur_workers w WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows, err := r.db.QueryxContext(ctx,
		`SELECT w.* FROM amaur_workers w WHERE `+where+
			fmt.Sprintf(` ORDER BY w.last_name ASC, w.first_name ASC LIMIT $%d OFFSET $%d`, idx, idx+1),
		args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var workers []*worker.Worker
	var ids []uuid.UUID
	for rows.Next() {
		var w worker.Worker
		if err := rows.StructScan(&w); err != nil {
			return nil, 0, err
		}
		workers = append(workers, &w)
		ids = append(ids, w.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Bulk-load specialties to avoid N+1.
	if len(ids) > 0 {
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
	_, err := r.db.ExecContext(ctx,
		`UPDATE amaur_workers SET user_id = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`,
		userID, workerID)
	return err
}

// ListSpecialties returns the active specialty catalog.
func (r *workerRepo) ListSpecialties(ctx context.Context) ([]worker.SpecialtyItem, error) {
	var items []worker.SpecialtyItem
	err := r.db.SelectContext(ctx, &items,
		`SELECT code, name FROM specialties WHERE is_active = true ORDER BY name`)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// CreateSpecialty inserts a new specialty into the catalog.
func (r *workerRepo) CreateSpecialty(ctx context.Context, item worker.SpecialtyItem) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO specialties (code, name) VALUES ($1, $2)`,
		item.Code, item.Name)
	return err
}

// DeleteSpecialty removes a specialty; returns an error if it is still referenced.
func (r *workerRepo) DeleteSpecialty(ctx context.Context, code string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM specialties WHERE code = $1`, code)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("specialty not found")
	}
	return nil
}

// GetWorkerSpecialties returns specialties linked to the given worker.
func (r *workerRepo) GetWorkerSpecialties(ctx context.Context, workerID uuid.UUID) ([]worker.SpecialtyItem, error) {
	var items []worker.SpecialtyItem
	err := r.db.SelectContext(ctx, &items, `
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

// SetWorkerSpecialties replaces all specialties for a worker atomically.
func (r *workerRepo) SetWorkerSpecialties(ctx context.Context, workerID uuid.UUID, codes []string, setBy uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM worker_specialties WHERE worker_id = $1`, workerID); err != nil {
		return err
	}
	for _, code := range codes {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO worker_specialties (id, worker_id, specialty_code, created_by)
			VALUES (uuid_generate_v4(), $1, $2, $3)
			ON CONFLICT (worker_id, specialty_code) DO NOTHING`,
			workerID, code, setBy); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// bulkLoadSpecialties fetches specialties for a set of worker IDs in one query.
func (r *workerRepo) bulkLoadSpecialties(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]worker.SpecialtyItem, error) {
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = id.String()
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT ws.worker_id, s.code, s.name
		FROM worker_specialties ws
		JOIN specialties s ON s.code = ws.specialty_code
		WHERE ws.worker_id = ANY($1::uuid[])
		ORDER BY ws.worker_id, s.name`,
		pq.Array(strs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]worker.SpecialtyItem)
	for rows.Next() {
		var wIDStr string
		var item worker.SpecialtyItem
		if err := rows.Scan(&wIDStr, &item.Code, &item.Name); err != nil {
			return nil, err
		}
		wID, _ := uuid.Parse(wIDStr)
		result[wID] = append(result[wID], item)
	}
	return result, rows.Err()
}

// ListAvailabilityRules returns all active availability rules for a worker.
func (r *workerRepo) ListAvailabilityRules(ctx context.Context, workerID uuid.UUID) ([]*worker.AvailabilityRule, error) {
	rules := make([]*worker.AvailabilityRule, 0)
	err := r.db.SelectContext(ctx, &rules, `
		SELECT id, worker_id, weekday,
		       TO_CHAR(start_time, 'HH24:MI') AS start_time,
		       TO_CHAR(end_time,   'HH24:MI') AS end_time,
		       is_active, created_at, created_by
		FROM worker_availability_rules
		WHERE worker_id = $1 AND is_active = TRUE
		ORDER BY weekday, start_time`, workerID)
	return rules, err
}

// ReplaceAvailabilityRules atomically replaces all rules for a worker.
func (r *workerRepo) ReplaceAvailabilityRules(ctx context.Context, workerID uuid.UUID, rules []*worker.AvailabilityRule) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM worker_availability_rules WHERE worker_id = $1`, workerID); err != nil {
		return err
	}
	for _, rule := range rules {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO worker_availability_rules
				(id, worker_id, weekday, start_time, end_time, is_active, created_by)
			VALUES ($1, $2, $3, $4::TIME, $5::TIME, TRUE, $6)`,
			rule.ID, workerID, rule.Weekday, rule.StartTime, rule.EndTime, rule.CreatedBy); err != nil {
			return err
		}
	}
	return tx.Commit()
}
