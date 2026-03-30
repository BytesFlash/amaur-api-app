package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"amaur/api/internal/domain/visit"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type visitRepo struct {
	db *sqlx.DB
}

func NewVisitRepository(db *sqlx.DB) visit.Repository {
	return &visitRepo{db: db}
}

func (r *visitRepo) Create(ctx context.Context, v *visit.Visit) error {
	v.ID = uuid.New()
	now := time.Now()
	v.CreatedAt = now
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO visits (id, company_id, branch_id, contract_id, status,
			scheduled_date, scheduled_start, scheduled_end,
			coordinator_user_id, general_notes, created_at, created_by)
		VALUES (:id, :company_id, :branch_id, :contract_id, :status,
			:scheduled_date, :scheduled_start, :scheduled_end,
			:coordinator_user_id, :general_notes, :created_at, :created_by)
	`, v)
	return err
}

func (r *visitRepo) FindByID(ctx context.Context, id uuid.UUID) (*visit.Visit, error) {
	v := &visit.Visit{}
	err := r.db.GetContext(ctx, v, `
		SELECT v.*, c.name AS company_name, c.fantasy_name AS company_fantasy_name
		FROM visits v
		JOIN companies c ON c.id = v.company_id
		WHERE v.id=$1`, id)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (r *visitRepo) Update(ctx context.Context, v *visit.Visit) error {
	now := time.Now()
	v.UpdatedAt = &now
	_, err := r.db.NamedExecContext(ctx, `
		UPDATE visits SET
			status=:status, scheduled_date=:scheduled_date,
			scheduled_start=:scheduled_start, scheduled_end=:scheduled_end,
			actual_start=:actual_start, actual_end=:actual_end,
			coordinator_user_id=:coordinator_user_id, general_notes=:general_notes,
			cancellation_reason=:cancellation_reason, internal_report=:internal_report,
			updated_at=:updated_at, updated_by=:updated_by
		WHERE id=:id
	`, v)
	return err
}

func (r *visitRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM visits WHERE id=$1`, id)
	return err
}

func (r *visitRepo) List(ctx context.Context, f visit.Filter, limit, offset int) ([]*visit.Visit, int64, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if f.CompanyID != nil {
		where = append(where, fmt.Sprintf("v.company_id=$%d", idx))
		args = append(args, *f.CompanyID)
		idx++
	}
	if f.PatientID != nil {
		where = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM care_sessions cs WHERE cs.visit_id = v.id AND cs.patient_id=$%d)", idx))
		args = append(args, *f.PatientID)
		idx++
	}
	if f.Status != "" {
		where = append(where, fmt.Sprintf("v.status=$%d", idx))
		args = append(args, f.Status)
		idx++
	}
	if f.DateFrom != nil {
		where = append(where, fmt.Sprintf("v.scheduled_date>=$%d", idx))
		args = append(args, *f.DateFrom)
		idx++
	}
	if f.DateTo != nil {
		where = append(where, fmt.Sprintf("v.scheduled_date<=$%d", idx))
		args = append(args, *f.DateTo)
		idx++
	}

	clause := strings.Join(where, " AND ")

	var total int64
	if err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM visits v JOIN companies c ON c.id = v.company_id WHERE `+clause, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows := []*visit.Visit{}
	if err := r.db.SelectContext(ctx, &rows,
		fmt.Sprintf(`
			SELECT v.*, c.name AS company_name, c.fantasy_name AS company_fantasy_name
			FROM visits v
			JOIN companies c ON c.id = v.company_id
			WHERE %s ORDER BY v.scheduled_date DESC LIMIT $%d OFFSET $%d`, clause, idx, idx+1),
		args...); err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *visitRepo) AssignWorkers(ctx context.Context, visitID uuid.UUID, workerIDs []uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM visit_workers WHERE visit_id=$1`, visitID); err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, wid := range workerIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO visit_workers (visit_id, worker_id, role_in_visit) VALUES ($1,$2,'profesional') ON CONFLICT DO NOTHING`, visitID, wid); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (r *visitRepo) ListWorkers(ctx context.Context, visitID uuid.UUID) ([]*visit.VisitWorker, error) {
	rows := []*visit.VisitWorker{}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT vw.visit_id, vw.worker_id, vw.role_in_visit,
			w.first_name, w.last_name, w.role_title
		FROM visit_workers vw
		JOIN amaur_workers w ON w.id = vw.worker_id
		WHERE vw.visit_id=$1`, visitID)
	return rows, err
}

func (r *visitRepo) HasPatientParticipation(ctx context.Context, visitID, patientID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS (SELECT 1 FROM care_sessions WHERE visit_id = $1 AND patient_id = $2)`,
		visitID, patientID)
	return exists, err
}
