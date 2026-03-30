package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"amaur/api/internal/domain/appointment"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type appointmentRepo struct {
	db *sqlx.DB
}

func NewAppointmentRepository(db *sqlx.DB) appointment.Repository {
	return &appointmentRepo{db: db}
}

const appointmentSelectSQL = `
	SELECT
		a.*,
		p.first_name || ' ' || p.last_name  AS patient_name,
		w.first_name || ' ' || w.last_name  AS worker_name,
		st.name                             AS service_type_name
	FROM appointments a
	JOIN patients p ON p.id = a.patient_id
	LEFT JOIN amaur_workers w ON w.id = a.worker_id
	JOIN service_types st ON st.id = a.service_type_id`

func (r *appointmentRepo) Create(ctx context.Context, a *appointment.Appointment) error {
	query := `
		INSERT INTO appointments (
			id, patient_id, worker_id, service_type_id, company_id,
			recurring_group_id, scheduled_at, duration_minutes, status, notes, created_by
		) VALUES (
			:id, :patient_id, :worker_id, :service_type_id, :company_id,
			:recurring_group_id, :scheduled_at, :duration_minutes, :status, :notes, :created_by
		)`
	_, err := r.db.NamedExecContext(ctx, query, a)
	return err
}

func (r *appointmentRepo) CreateBatch(ctx context.Context, batch []*appointment.Appointment) error {
	if len(batch) == 0 {
		return nil
	}
	query := `
		INSERT INTO appointments (
			id, patient_id, worker_id, service_type_id, company_id,
			recurring_group_id, scheduled_at, duration_minutes, status, notes, created_by
		) VALUES `
	vals := make([]string, 0, len(batch))
	args := make([]interface{}, 0, len(batch)*11)
	for i, a := range batch {
		n := i * 11
		vals = append(vals, fmt.Sprintf(
			"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8, n+9, n+10, n+11,
		))
		args = append(args,
			a.ID, a.PatientID, a.WorkerID, a.ServiceTypeID, a.CompanyID,
			a.RecurringGroupID, a.ScheduledAt, a.DurationMinutes, a.Status, a.Notes, a.CreatedBy,
		)
	}
	_, err := r.db.ExecContext(ctx, query+strings.Join(vals, ","), args...)
	return err
}

func (r *appointmentRepo) FindByID(ctx context.Context, id uuid.UUID) (*appointment.Appointment, error) {
	var a appointment.Appointment
	err := r.db.GetContext(ctx, &a, appointmentSelectSQL+` WHERE a.id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *appointmentRepo) Update(ctx context.Context, a *appointment.Appointment) error {
	_, err := r.db.NamedExecContext(ctx, `
		UPDATE appointments SET
			worker_id        = :worker_id,
			service_type_id  = :service_type_id,
			company_id       = :company_id,
			scheduled_at     = :scheduled_at,
			duration_minutes = :duration_minutes,
			status           = :status,
			notes            = :notes,
			updated_at       = NOW()
		WHERE id = :id`, a)
	return err
}

func (r *appointmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM appointments WHERE id = $1`, id)
	return err
}

func (r *appointmentRepo) List(ctx context.Context, f appointment.Filter, limit, offset int) ([]*appointment.Appointment, int64, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if f.PatientID != nil {
		where = append(where, fmt.Sprintf("a.patient_id = $%d", idx))
		args = append(args, *f.PatientID)
		idx++
	}
	if f.WorkerID != nil {
		where = append(where, fmt.Sprintf("a.worker_id = $%d", idx))
		args = append(args, *f.WorkerID)
		idx++
	}
	if f.CompanyID != nil {
		where = append(where, fmt.Sprintf("a.company_id = $%d", idx))
		args = append(args, *f.CompanyID)
		idx++
	}
	if f.Status != "" {
		where = append(where, fmt.Sprintf("a.status = $%d", idx))
		args = append(args, f.Status)
		idx++
	}
	if f.DateFrom != nil {
		where = append(where, fmt.Sprintf("a.scheduled_at >= $%d", idx))
		args = append(args, f.DateFrom.Format(time.RFC3339))
		idx++
	}
	if f.DateTo != nil {
		where = append(where, fmt.Sprintf("a.scheduled_at <= $%d", idx))
		args = append(args, f.DateTo.Add(24*time.Hour).Format(time.RFC3339))
		idx++
	}

	whereClause := strings.Join(where, " AND ")

	var total int64
	countSQL := `SELECT COUNT(*) FROM appointments a JOIN patients p ON p.id = a.patient_id LEFT JOIN amaur_workers w ON w.id = a.worker_id JOIN service_types st ON st.id = a.service_type_id WHERE ` + whereClause
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	listSQL := appointmentSelectSQL + ` WHERE ` + whereClause +
		fmt.Sprintf(` ORDER BY a.scheduled_at DESC LIMIT $%d OFFSET $%d`, idx, idx+1)

	rows, err := r.db.QueryxContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*appointment.Appointment
	for rows.Next() {
		var a appointment.Appointment
		if err := rows.StructScan(&a); err != nil {
			return nil, 0, err
		}
		items = append(items, &a)
	}
	return items, total, rows.Err()
}
