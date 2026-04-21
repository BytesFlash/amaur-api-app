package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"amaur/api/internal/domain/appointment"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type appointmentRepo struct {
	db *gorm.DB
}

func NewAppointmentRepository(db *gorm.DB) appointment.Repository {
	return &appointmentRepo{db: db}
}

const appointmentSelectSQL = `
	SELECT
		a.*,
		p.first_name || ' ' || p.last_name  AS patient_name,
		w.first_name || ' ' || w.last_name  AS worker_name,
		st.name                             AS service_type_name,
		c.name                              AS company_name
	FROM appointments a
	JOIN patients p ON p.id = a.patient_id
	LEFT JOIN amaur_workers w ON w.id = a.worker_id
	JOIN service_types st ON st.id = a.service_type_id
	LEFT JOIN companies c ON c.id = a.company_id`

func (r *appointmentRepo) Create(ctx context.Context, a *appointment.Appointment) error {
	return rawExec(ctx, r.db, `
		INSERT INTO appointments (
			id, patient_id, worker_id, service_type_id, company_id,
			recurring_group_id, scheduled_at, duration_minutes, status, notes,
			chief_complaint, subjective, objective, assessment, "plan",
			follow_up_required, follow_up_notes, follow_up_date, created_by
		) VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,$9,$10,
			$11,$12,$13,$14,$15,
			$16,$17,$18,$19
		)`,
		a.ID, a.PatientID, a.WorkerID, a.ServiceTypeID, a.CompanyID,
		a.RecurringGroupID, a.ScheduledAt, a.DurationMinutes, a.Status, a.Notes,
		a.ChiefComplaint, a.Subjective, a.Objective, a.Assessment, a.Plan,
		a.FollowUpRequired, a.FollowUpNotes, a.FollowUpDate, a.CreatedBy)
}

func (r *appointmentRepo) CreateBatch(ctx context.Context, batch []*appointment.Appointment) error {
	if len(batch) == 0 {
		return nil
	}
	query := `
		INSERT INTO appointments (
			id, patient_id, worker_id, service_type_id, company_id,
			recurring_group_id, scheduled_at, duration_minutes, status, notes,
			chief_complaint, subjective, objective, assessment, "plan",
			follow_up_required, follow_up_notes, follow_up_date, created_by
		) VALUES `
	vals := make([]string, 0, len(batch))
	args := make([]interface{}, 0, len(batch)*19)
	for i, a := range batch {
		n := i * 19
		vals = append(vals, fmt.Sprintf(
			"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8, n+9, n+10, n+11, n+12, n+13, n+14, n+15, n+16, n+17, n+18, n+19,
		))
		args = append(args,
			a.ID, a.PatientID, a.WorkerID, a.ServiceTypeID, a.CompanyID,
			a.RecurringGroupID, a.ScheduledAt, a.DurationMinutes, a.Status, a.Notes,
			a.ChiefComplaint, a.Subjective, a.Objective, a.Assessment, a.Plan,
			a.FollowUpRequired, a.FollowUpNotes, a.FollowUpDate, a.CreatedBy,
		)
	}
	return rawExec(ctx, r.db, query+strings.Join(vals, ","), args...)
}

func (r *appointmentRepo) FindByID(ctx context.Context, id uuid.UUID) (*appointment.Appointment, error) {
	var a appointment.Appointment
	err := rawGet(ctx, r.db, &a, appointmentSelectSQL+` WHERE a.id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *appointmentRepo) Update(ctx context.Context, a *appointment.Appointment) error {
	return rawExec(ctx, r.db, `
		UPDATE appointments SET
			worker_id        = $1,
			service_type_id  = $2,
			company_id       = $3,
			scheduled_at     = $4,
			duration_minutes = $5,
			status           = $6,
			notes            = $7,
			chief_complaint  = $8,
			subjective       = $9,
			objective        = $10,
			assessment       = $11,
			"plan"           = $12,
			follow_up_required = $13,
			follow_up_notes  = $14,
			follow_up_date   = $15,
			updated_at       = NOW()
		WHERE id = $16`,
		a.WorkerID, a.ServiceTypeID, a.CompanyID, a.ScheduledAt, a.DurationMinutes, a.Status, a.Notes,
		a.ChiefComplaint, a.Subjective, a.Objective, a.Assessment, a.Plan, a.FollowUpRequired, a.FollowUpNotes, a.FollowUpDate, a.ID)
}

func (r *appointmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return rawExec(ctx, r.db, `DELETE FROM appointments WHERE id = $1`, id)
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

	var totalRow struct {
		Count int64 `gorm:"column:count"`
	}
	countSQL := `SELECT COUNT(*) AS count FROM appointments a JOIN patients p ON p.id = a.patient_id LEFT JOIN amaur_workers w ON w.id = a.worker_id JOIN service_types st ON st.id = a.service_type_id WHERE ` + whereClause
	if err := rawGet(ctx, r.db, &totalRow, countSQL, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	listSQL := appointmentSelectSQL + ` WHERE ` + whereClause +
		fmt.Sprintf(` ORDER BY a.scheduled_at DESC LIMIT $%d OFFSET $%d`, idx, idx+1)

	var items []*appointment.Appointment
	if err := rawSelectPtr(ctx, r.db, &items, listSQL, args...); err != nil {
		return nil, 0, err
	}
	return items, totalRow.Count, nil
}

func (r *appointmentRepo) HasWorkerConflict(ctx context.Context, workerID uuid.UUID, scheduledAt time.Time, durationMinutes int, excludeID *uuid.UUID) (bool, error) {
	if durationMinutes <= 0 {
		durationMinutes = 60
	}

	query := `
		SELECT EXISTS (
			SELECT 1
			FROM appointments a
			WHERE a.worker_id = $1
			  AND a.status IN ('requested', 'confirmed', 'in_progress', 'scheduled')
			  AND a.scheduled_at < ($2::timestamptz + ($3::int * INTERVAL '1 minute'))
			  AND (a.scheduled_at + (COALESCE(a.duration_minutes, 60) * INTERVAL '1 minute')) > $2::timestamptz
			  AND ($4::uuid IS NULL OR a.id <> $4)
		) AS exists
	`

	var excluded sql.NullString
	if excludeID != nil {
		excluded.Valid = true
		excluded.String = excludeID.String()
	}

	var row struct {
		Exists bool `gorm:"column:exists"`
	}
	err := rawGet(ctx, r.db, &row, query, workerID, scheduledAt, durationMinutes, excluded)
	return row.Exists, err
}
