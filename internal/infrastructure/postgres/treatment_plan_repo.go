package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"amaur/api/internal/domain/treatmentplan"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type treatmentPlanRepo struct {
	db *gorm.DB
}

func NewTreatmentPlanRepository(db *gorm.DB) treatmentplan.Repository {
	return &treatmentPlanRepo{db: db}
}

const treatmentPlanSelectSQL = `
	SELECT
		tp.*,
		p.first_name || ' ' || p.last_name   AS patient_name,
		w.first_name || ' ' || w.last_name   AS professional_name,
		st.name                              AS service_type_name
	FROM treatment_plans tp
	JOIN patients p ON p.id = tp.patient_id
	LEFT JOIN amaur_workers w ON w.id = tp.professional_id
	LEFT JOIN service_types st ON st.id = tp.service_type_id`

func (r *treatmentPlanRepo) Create(ctx context.Context, p *treatmentplan.TreatmentPlan) error {
	return rawExec(ctx, r.db, `
		INSERT INTO treatment_plans (
			id, patient_id, professional_id, service_type_id,
			title, objective, total_sessions, completed_sessions,
			frequency_type, frequency_interval,
			start_date, estimated_end_date, status, notes, created_by
		) VALUES (
			$1,$2,$3,$4,
			$5,$6,$7,$8,
			$9,$10,
			$11,$12,$13,$14,$15
		)`,
		p.ID, p.PatientID, p.ProfessionalID, p.ServiceTypeID,
		p.Title, p.Objective, p.TotalSessions, p.CompletedSessions,
		p.FrequencyType, p.FrequencyInterval,
		p.StartDate, p.EstimatedEndDate, p.Status, p.Notes, p.CreatedBy)
}

func (r *treatmentPlanRepo) FindByID(ctx context.Context, id uuid.UUID) (*treatmentplan.TreatmentPlan, error) {
	var p treatmentplan.TreatmentPlan
	err := rawGet(ctx, r.db, &p, treatmentPlanSelectSQL+` WHERE tp.id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *treatmentPlanRepo) Update(ctx context.Context, p *treatmentplan.TreatmentPlan) error {
	return rawExec(ctx, r.db, `
		UPDATE treatment_plans SET
			professional_id    = $1,
			service_type_id    = $2,
			title              = $3,
			objective          = $4,
			total_sessions     = $5,
			frequency_type     = $6,
			frequency_interval = $7,
			start_date         = $8,
			estimated_end_date = $9,
			status             = $10,
			notes              = $11,
			updated_at         = NOW()
		WHERE id = $12`,
		p.ProfessionalID, p.ServiceTypeID, p.Title, p.Objective,
		p.TotalSessions, p.FrequencyType, p.FrequencyInterval,
		p.StartDate, p.EstimatedEndDate, p.Status, p.Notes, p.ID)
}

func (r *treatmentPlanRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return rawExec(ctx, r.db, `DELETE FROM treatment_plans WHERE id = $1`, id)
}

func (r *treatmentPlanRepo) List(ctx context.Context, f treatmentplan.Filter, limit, offset int) ([]*treatmentplan.TreatmentPlan, int64, error) {
	db := r.db.WithContext(ctx).
		Table("treatment_plans tp").
		Joins("JOIN patients p ON p.id = tp.patient_id").
		Joins("LEFT JOIN amaur_workers w ON w.id = tp.professional_id").
		Joins("LEFT JOIN service_types st ON st.id = tp.service_type_id")

	if f.PatientID != nil {
		db = db.Where("tp.patient_id = ?", *f.PatientID)
	}
	if f.ProfessionalID != nil {
		db = db.Where("tp.professional_id = ?", *f.ProfessionalID)
	}
	if f.Status != "" {
		db = db.Where("tp.status = ?", f.Status)
	}
	if f.ServiceTypeID != nil {
		db = db.Where("tp.service_type_id = ?", *f.ServiceTypeID)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []*treatmentplan.TreatmentPlan
	err := db.
		Select(`tp.*,
			p.first_name || ' ' || p.last_name  AS patient_name,
			w.first_name || ' ' || w.last_name  AS professional_name,
			st.name                             AS service_type_name`).
		Order("tp.created_at DESC").
		Limit(limit).Offset(offset).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *treatmentPlanRepo) RecalculateCompleted(ctx context.Context, planID uuid.UUID) error {
	return rawExec(ctx, r.db, `
		UPDATE treatment_plans
		SET completed_sessions = (
			SELECT COUNT(*)
			FROM appointments
			WHERE treatment_plan_id = $1
			  AND counts_as_session = TRUE
			  AND status IN ('completed', 'no_show')
		),
		updated_at = NOW()
		WHERE id = $1`, planID)
}

func (r *treatmentPlanRepo) GetAlerts(ctx context.Context, professionalID *uuid.UUID) ([]*treatmentplan.Alert, error) {
	conds := []string{"tp.status = 'active'"}
	args := []interface{}{}
	argIdx := 1

	if professionalID != nil {
		conds = append(conds, fmt.Sprintf("tp.professional_id = $%d", argIdx))
		args = append(args, *professionalID)
		argIdx++
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	// Fecha de último appointment completado por plan
	query := fmt.Sprintf(`
		SELECT
			CASE
				WHEN tp.completed_sessions >= tp.total_sessions THEN 'plan_complete'
				WHEN tp.completed_sessions = tp.total_sessions - 1 THEN 'last_session_pending'
				WHEN last_appt.last_date IS NOT NULL
				     AND (NOW()::date - last_appt.last_date) > (tp.frequency_interval * 2) THEN 'overdue'
				ELSE NULL
			END                          AS type,
			tp.id                        AS treatment_plan_id,
			tp.patient_id,
			p.first_name || ' ' || p.last_name AS patient_name,
			tp.title                     AS plan_title,
			''                           AS message,
			last_appt.last_date          AS since_date,
			(tp.total_sessions - tp.completed_sessions) AS sessions_left
		FROM treatment_plans tp
		JOIN patients p ON p.id = tp.patient_id
		LEFT JOIN (
			SELECT treatment_plan_id, MAX(scheduled_at::date) AS last_date
			FROM appointments
			WHERE counts_as_session = TRUE
			  AND status IN ('completed','no_show')
			GROUP BY treatment_plan_id
		) last_appt ON last_appt.treatment_plan_id = tp.id
		%s
		HAVING CASE
			WHEN tp.completed_sessions >= tp.total_sessions THEN 'plan_complete'
			WHEN tp.completed_sessions = tp.total_sessions - 1 THEN 'last_session_pending'
			WHEN last_appt.last_date IS NOT NULL
			     AND (NOW()::date - last_appt.last_date) > (tp.frequency_interval * 2) THEN 'overdue'
			ELSE NULL
		END IS NOT NULL
		ORDER BY tp.patient_id`, where)

	var rows []struct {
		Type            string     `gorm:"column:type"`
		TreatmentPlanID uuid.UUID  `gorm:"column:treatment_plan_id"`
		PatientID       uuid.UUID  `gorm:"column:patient_id"`
		PatientName     *string    `gorm:"column:patient_name"`
		PlanTitle       string     `gorm:"column:plan_title"`
		Message         string     `gorm:"column:message"`
		SinceDate       *time.Time `gorm:"column:since_date"`
		SessionsLeft    *int       `gorm:"column:sessions_left"`
	}
	if err := rawSelect(ctx, r.db, &rows, query, args...); err != nil {
		return nil, err
	}
	alerts := make([]*treatmentplan.Alert, 0, len(rows))
	for _, row := range rows {
		row := row
		alert := &treatmentplan.Alert{
			Type:            row.Type,
			TreatmentPlanID: row.TreatmentPlanID,
			PatientID:       row.PatientID,
			PatientName:     row.PatientName,
			PlanTitle:       row.PlanTitle,
			SinceDate:       row.SinceDate,
			SessionsLeft:    row.SessionsLeft,
		}
		switch row.Type {
		case "plan_complete":
			alert.Message = "El plan de tratamiento ha sido completado."
		case "last_session_pending":
			alert.Message = "Queda 1 sesión para completar el plan."
		case "overdue":
			alert.Message = "El paciente no ha asistido en más del doble del intervalo esperado."
		}
		alerts = append(alerts, alert)
	}
	return alerts, nil
}
