package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"amaur/api/internal/domain/visit"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type visitRepo struct {
	db *gorm.DB
}

func NewVisitRepository(db *gorm.DB) visit.Repository {
	return &visitRepo{db: db}
}

func (r *visitRepo) Create(ctx context.Context, v *visit.Visit) error {
	v.ID = uuid.New()
	now := time.Now()
	v.CreatedAt = now
	return rawExec(ctx, r.db, `
		INSERT INTO visits (id, company_id, branch_id, contract_id, status,
			scheduled_date, scheduled_start, scheduled_end,
			coordinator_user_id, general_notes, created_at, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`, v.ID, v.CompanyID, v.BranchID, v.ContractID, v.Status,
		v.ScheduledDate, v.ScheduledStart, v.ScheduledEnd,
		v.CoordinatorUserID, v.GeneralNotes, v.CreatedAt, v.CreatedBy)
}

func (r *visitRepo) FindByID(ctx context.Context, id uuid.UUID) (*visit.Visit, error) {
	v := &visit.Visit{}
	err := rawGet(ctx, r.db, v, `
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
	return rawExec(ctx, r.db, `
		UPDATE visits SET
			status=$1, scheduled_date=$2,
			scheduled_start=$3, scheduled_end=$4,
			actual_start=$5, actual_end=$6,
			coordinator_user_id=$7, general_notes=$8,
			cancellation_reason=$9, internal_report=$10,
			updated_at=$11, updated_by=$12
		WHERE id=$13
	`, v.Status, v.ScheduledDate,
		v.ScheduledStart, v.ScheduledEnd,
		v.ActualStart, v.ActualEnd,
		v.CoordinatorUserID, v.GeneralNotes,
		v.CancellationReason, v.InternalReport,
		v.UpdatedAt, v.UpdatedBy, v.ID)
}

func (r *visitRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return rawExec(ctx, r.db, `DELETE FROM visits WHERE id=$1`, id)
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

	var totalRow struct {
		Count int64 `gorm:"column:count"`
	}
	if err := rawGet(ctx, r.db, &totalRow,
		`SELECT COUNT(*) AS count FROM visits v JOIN companies c ON c.id = v.company_id WHERE `+clause, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows := []*visit.Visit{}
	if err := rawSelectPtr(ctx, r.db, &rows, fmt.Sprintf(`
			SELECT v.*, c.name AS company_name, c.fantasy_name AS company_fantasy_name
			FROM visits v
			JOIN companies c ON c.id = v.company_id
			WHERE %s ORDER BY v.scheduled_date DESC LIMIT $%d OFFSET $%d`, clause, idx, idx+1),
		args...); err != nil {
		return nil, 0, err
	}

	return rows, totalRow.Count, nil
}

func (r *visitRepo) AssignWorkers(ctx context.Context, visitID uuid.UUID, workerIDs []uuid.UUID) error {
	return withTx(ctx, r.db, func(tx *gorm.DB) error {
		if err := rawExec(ctx, tx, `DELETE FROM visit_workers WHERE visit_id=$1`, visitID); err != nil {
			return err
		}
		for _, wid := range workerIDs {
			if err := rawExec(ctx, tx, `INSERT INTO visit_workers (visit_id, worker_id, role_in_visit) VALUES ($1,$2,'profesional') ON CONFLICT DO NOTHING`, visitID, wid); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *visitRepo) ListWorkers(ctx context.Context, visitID uuid.UUID) ([]*visit.VisitWorker, error) {
	rows := []*visit.VisitWorker{}
	err := rawSelectPtr(ctx, r.db, &rows, `
		SELECT vw.visit_id, vw.worker_id, vw.role_in_visit,
			w.first_name, w.last_name, w.role_title
		FROM visit_workers vw
		JOIN amaur_workers w ON w.id = vw.worker_id
		WHERE vw.visit_id=$1`, visitID)
	return rows, err
}

func (r *visitRepo) HasPatientParticipation(ctx context.Context, visitID, patientID uuid.UUID) (bool, error) {
	var row struct {
		Exists bool `gorm:"column:exists"`
	}
	err := rawGet(ctx, r.db, &row,
		`SELECT EXISTS (SELECT 1 FROM care_sessions WHERE visit_id = $1 AND patient_id = $2) AS exists`,
		visitID, patientID)
	return row.Exists, err
}
