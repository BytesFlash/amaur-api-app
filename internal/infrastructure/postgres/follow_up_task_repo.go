package postgres

import (
	"context"
	"time"

	"amaur/api/internal/domain/followuptask"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type followUpTaskRepo struct {
	db *gorm.DB
}

func NewFollowUpTaskRepository(db *gorm.DB) followuptask.Repository {
	return &followUpTaskRepo{db: db}
}

const followUpTaskSelectSQL = `
	SELECT
		ft.*,
		p.first_name  || ' ' || p.last_name AS patient_name,
		w.first_name  || ' ' || w.last_name AS professional_name
	FROM follow_up_tasks ft
	JOIN patients p ON p.id = ft.patient_id
	LEFT JOIN amaur_workers w ON w.id = ft.professional_id`

func (r *followUpTaskRepo) Create(ctx context.Context, t *followuptask.FollowUpTask) error {
	return rawExec(ctx, r.db, `
		INSERT INTO follow_up_tasks (
			id, patient_id, treatment_plan_id, appointment_id, professional_id,
			title, description, due_date, status, priority, created_by
		) VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,$9,$10,$11
		)`,
		t.ID, t.PatientID, t.TreatmentPlanID, t.AppointmentID, t.ProfessionalID,
		t.Title, t.Description, t.DueDate, t.Status, t.Priority, t.CreatedBy)
}

func (r *followUpTaskRepo) FindByID(ctx context.Context, id uuid.UUID) (*followuptask.FollowUpTask, error) {
	var t followuptask.FollowUpTask
	err := rawGet(ctx, r.db, &t, followUpTaskSelectSQL+` WHERE ft.id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *followUpTaskRepo) Update(ctx context.Context, t *followuptask.FollowUpTask) error {
	return rawExec(ctx, r.db, `
		UPDATE follow_up_tasks SET
			title           = $1,
			description     = $2,
			due_date        = $3,
			status          = $4,
			priority        = $5,
			professional_id = $6,
			updated_at      = NOW()
		WHERE id = $7`,
		t.Title, t.Description, t.DueDate, t.Status, t.Priority, t.ProfessionalID, t.ID)
}

func (r *followUpTaskRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return rawExec(ctx, r.db, `DELETE FROM follow_up_tasks WHERE id = $1`, id)
}

func (r *followUpTaskRepo) List(ctx context.Context, f followuptask.Filter, limit, offset int) ([]*followuptask.FollowUpTask, int64, error) {
	db := r.db.WithContext(ctx).
		Table("follow_up_tasks ft").
		Joins("JOIN patients p ON p.id = ft.patient_id").
		Joins("LEFT JOIN amaur_workers w ON w.id = ft.professional_id")

	if f.PatientID != nil {
		db = db.Where("ft.patient_id = ?", *f.PatientID)
	}
	if f.ProfessionalID != nil {
		db = db.Where("ft.professional_id = ?", *f.ProfessionalID)
	}
	if f.Status != "" {
		db = db.Where("ft.status = ?", f.Status)
	}
	if f.DueBefore != nil {
		db = db.Where("ft.due_date <= ?", f.DueBefore.Format(time.DateOnly))
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []*followuptask.FollowUpTask
	err := db.
		Select(`ft.*,
			p.first_name || ' ' || p.last_name AS patient_name,
			w.first_name || ' ' || w.last_name AS professional_name`).
		Order("ft.due_date ASC").
		Limit(limit).Offset(offset).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
