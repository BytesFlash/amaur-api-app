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
	LEFT JOIN service_types st ON st.id = a.service_type_id
	LEFT JOIN companies c ON c.id = a.company_id`

func (r *appointmentRepo) Create(ctx context.Context, a *appointment.Appointment) error {
	return rawExec(ctx, r.db, `
		INSERT INTO appointments (
			id, patient_id, worker_id, service_type_id, company_id,
			recurring_group_id, scheduled_at, duration_minutes, status, notes,
			chief_complaint, subjective, objective, assessment, "plan",
			follow_up_required, follow_up_notes, follow_up_date,
			treatment_plan_id, session_number, counts_as_session,
			created_by
		) VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,$9,$10,
			$11,$12,$13,$14,$15,
			$16,$17,$18,
			$19,$20,$21,
			$22
		)`,
		a.ID, a.PatientID, a.WorkerID, a.ServiceTypeID, a.CompanyID,
		a.RecurringGroupID, a.ScheduledAt, a.DurationMinutes, a.Status, a.Notes,
		a.ChiefComplaint, a.Subjective, a.Objective, a.Assessment, a.Plan,
		a.FollowUpRequired, a.FollowUpNotes, a.FollowUpDate,
		a.TreatmentPlanID, a.SessionNumber, a.CountsAsSession,
		a.CreatedBy)
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
			follow_up_required, follow_up_notes, follow_up_date,
			treatment_plan_id, session_number, counts_as_session,
			created_by
		) VALUES `
	const cols = 22
	vals := make([]string, 0, len(batch))
	args := make([]interface{}, 0, len(batch)*cols)
	for i, a := range batch {
		n := i * cols
		vals = append(vals, fmt.Sprintf(
			"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8, n+9, n+10, n+11,
			n+12, n+13, n+14, n+15, n+16, n+17, n+18, n+19, n+20, n+21, n+22,
		))
		args = append(args,
			a.ID, a.PatientID, a.WorkerID, a.ServiceTypeID, a.CompanyID,
			a.RecurringGroupID, a.ScheduledAt, a.DurationMinutes, a.Status, a.Notes,
			a.ChiefComplaint, a.Subjective, a.Objective, a.Assessment, a.Plan,
			a.FollowUpRequired, a.FollowUpNotes, a.FollowUpDate,
			a.TreatmentPlanID, a.SessionNumber, a.CountsAsSession,
			a.CreatedBy,
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
			worker_id          = $1,
			service_type_id    = $2,
			company_id         = $3,
			scheduled_at       = $4,
			duration_minutes   = $5,
			status             = $6,
			notes              = $7,
			chief_complaint    = $8,
			subjective         = $9,
			objective          = $10,
			assessment         = $11,
			"plan"             = $12,
			follow_up_required = $13,
			follow_up_notes    = $14,
			follow_up_date     = $15,
			care_session_id    = $16,
			treatment_plan_id  = $17,
			session_number     = $18,
			counts_as_session  = $19,
			updated_at         = NOW()
		WHERE id = $20`,
		a.WorkerID, a.ServiceTypeID, a.CompanyID, a.ScheduledAt, a.DurationMinutes, a.Status, a.Notes,
		a.ChiefComplaint, a.Subjective, a.Objective, a.Assessment, a.Plan, a.FollowUpRequired, a.FollowUpNotes,
		a.FollowUpDate, a.CareSessionID, a.TreatmentPlanID, a.SessionNumber, a.CountsAsSession, a.ID)
}

func (r *appointmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return rawExec(ctx, r.db, `DELETE FROM appointments WHERE id = $1`, id)
}

func (r *appointmentRepo) List(ctx context.Context, f appointment.Filter, limit, offset int) ([]*appointment.Appointment, int64, error) {
	db := r.db.WithContext(ctx).
		Table("appointments a").
		Joins("JOIN patients p ON p.id = a.patient_id").
		Joins("LEFT JOIN amaur_workers w ON w.id = a.worker_id").
		Joins("LEFT JOIN service_types st ON st.id = a.service_type_id").
		Joins("LEFT JOIN companies c ON c.id = a.company_id")

	if f.PatientID != nil {
		db = db.Where("a.patient_id = ?", *f.PatientID)
	}
	if f.WorkerID != nil {
		db = db.Where("a.worker_id = ?", *f.WorkerID)
	}
	if f.CompanyID != nil {
		db = db.Where("a.company_id = ?", *f.CompanyID)
	}
	if f.TreatmentPlanID != nil {
		db = db.Where("a.treatment_plan_id = ?", *f.TreatmentPlanID)
	}
	if f.Status != "" {
		db = db.Where("a.status = ?", f.Status)
	}
	if f.DateFrom != nil {
		db = db.Where("a.scheduled_at >= ?", f.DateFrom.Format(time.RFC3339))
	}
	if f.DateTo != nil {
		db = db.Where("a.scheduled_at <= ?", f.DateTo.Add(24*time.Hour).Format(time.RFC3339))
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []*appointment.Appointment
	err := db.
		Select(`a.*,
			p.first_name || ' ' || p.last_name AS patient_name,
			w.first_name || ' ' || w.last_name AS worker_name,
			st.name                            AS service_type_name,
			c.name                             AS company_name`).
		Order("a.scheduled_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
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

// CountActiveSessions returns the number of non-terminal appointments linked to a treatment plan.
func (r *appointmentRepo) CountActiveSessions(ctx context.Context, planID uuid.UUID) (int, error) {
	var row struct {
		Count int `gorm:"column:count"`
	}
	err := rawGet(ctx, r.db, &row, `
		SELECT COUNT(*) AS count
		FROM appointments
		WHERE treatment_plan_id = $1
		  AND status NOT IN ('cancelled')`, planID)
	if err != nil {
		return 0, err
	}
	return row.Count, nil
}
