package postgres

import (
	"context"
	"time"

	"amaur/api/internal/domain/caresession"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type careSessionRepo struct {
	db *gorm.DB
}

func NewCareSessionRepository(db *gorm.DB) caresession.Repository {
	return &careSessionRepo{db: db}
}

func (r *careSessionRepo) Create(ctx context.Context, cs *caresession.CareSession) error {
	cs.ID = uuid.New()
	cs.CreatedAt = time.Now()
	return rawExec(ctx, r.db, `
		INSERT INTO care_sessions (
			id, visit_id, patient_id, worker_id, service_type_id, company_id,
			contract_service_id, session_type, session_date, session_time,
			duration_minutes, status, chief_complaint, subjective, objective,
			assessment, plan, notes, follow_up_required, created_at, created_by
		) VALUES (
			$1,$2,$3,$4,$5,$6,
			$7,$8,$9,$10,
			$11,$12,$13,$14,$15,
			$16,$17,$18,$19,$20,$21
		)`, cs.ID, cs.VisitID, cs.PatientID, cs.WorkerID, cs.ServiceTypeID, cs.CompanyID,
		cs.ContractServiceID, cs.SessionType, cs.SessionDate, cs.SessionTime,
		cs.DurationMinutes, cs.Status, cs.ChiefComplaint, cs.Subjective, cs.Objective,
		cs.Assessment, cs.Plan, cs.Notes, cs.FollowUpRequired, cs.CreatedAt, cs.CreatedBy)
}

func (r *careSessionRepo) FindByID(ctx context.Context, id uuid.UUID) (*caresession.CareSession, error) {
	var cs caresession.CareSession
	err := r.db.WithContext(ctx).
		Table("care_sessions cs").
		Joins("LEFT JOIN patients p ON p.id = cs.patient_id").
		Joins("LEFT JOIN amaur_workers w ON w.id = cs.worker_id").
		Joins("LEFT JOIN service_types st ON st.id = cs.service_type_id").
		Joins("LEFT JOIN companies c ON c.id = cs.company_id").
		Select(`cs.*,
			p.first_name AS patient_first_name, p.last_name AS patient_last_name,
			w.first_name AS worker_first_name, w.last_name AS worker_last_name,
			st.name AS service_type_name, c.name AS company_name`).
		Where("cs.id = ?", id).
		Scan(&cs).Error
	if err != nil {
		return nil, err
	}
	if cs.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &cs, nil
}

func (r *careSessionRepo) Update(ctx context.Context, cs *caresession.CareSession) error {
	now := time.Now()
	cs.UpdatedAt = &now
	return rawExec(ctx, r.db, `
		UPDATE care_sessions SET
			status=$1, session_date=$2, session_time=$3,
			duration_minutes=$4, chief_complaint=$5,
			subjective=$6, objective=$7, assessment=$8,
			plan=$9, notes=$10, follow_up_required=$11,
			follow_up_status=$12, follow_up_date=$13,
			follow_up_notes=$14, updated_at=$15, updated_by=$16
		WHERE id=$17`, cs.Status, cs.SessionDate, cs.SessionTime,
		cs.DurationMinutes, cs.ChiefComplaint,
		cs.Subjective, cs.Objective, cs.Assessment,
		cs.Plan, cs.Notes, cs.FollowUpRequired,
		cs.FollowUpStatus, cs.FollowUpDate,
		cs.FollowUpNotes, cs.UpdatedAt, cs.UpdatedBy, cs.ID)
}

func (r *careSessionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return rawExec(ctx, r.db, `DELETE FROM care_sessions WHERE id=$1`, id)
}

func (r *careSessionRepo) List(ctx context.Context, f caresession.Filter, limit, offset int) ([]*caresession.CareSession, int64, error) {
	db := r.db.WithContext(ctx).
		Table("care_sessions cs").
		Joins("LEFT JOIN patients p ON p.id = cs.patient_id").
		Joins("LEFT JOIN amaur_workers w ON w.id = cs.worker_id").
		Joins("LEFT JOIN service_types st ON st.id = cs.service_type_id").
		Joins("LEFT JOIN companies c ON c.id = cs.company_id")

	if f.PatientID != nil {
		db = db.Where("cs.patient_id = ?", *f.PatientID)
	}
	if f.WorkerID != nil {
		db = db.Where("cs.worker_id = ?", *f.WorkerID)
	}
	if f.CompanyID != nil {
		db = db.Where("cs.company_id = ?", *f.CompanyID)
	}
	if f.VisitID != nil {
		db = db.Where("cs.visit_id = ?", *f.VisitID)
	}
	if f.SessionType != "" {
		db = db.Where("cs.session_type = ?", f.SessionType)
	}
	if f.Status != "" {
		db = db.Where("cs.status = ?", f.Status)
	}
	if f.DateFrom != nil {
		db = db.Where("cs.session_date >= ?", *f.DateFrom)
	}
	if f.DateTo != nil {
		db = db.Where("cs.session_date <= ?", *f.DateTo)
	}

	// COUNT uses same JOINs and WHERE as the data query — consistent totals.
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []*caresession.CareSession
	err := db.
		Select(`cs.*,
			p.first_name AS patient_first_name, p.last_name AS patient_last_name,
			w.first_name AS worker_first_name, w.last_name AS worker_last_name,
			st.name AS service_type_name, c.name AS company_name`).
		Order("cs.session_date DESC, cs.session_time DESC NULLS LAST").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *careSessionRepo) CreateGroupSession(ctx context.Context, gs *caresession.GroupSession) error {
	gs.ID = uuid.New()
	gs.CreatedAt = time.Now()
	return rawExec(ctx, r.db, `
		INSERT INTO group_sessions (
			id, visit_id, service_type_id, worker_id, attendee_count,
			session_date, session_time, duration_minutes, notes, created_at, created_by
		) VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,$9,$10,$11
		)`, gs.ID, gs.VisitID, gs.ServiceTypeID, gs.WorkerID, gs.AttendeeCount,
		gs.SessionDate, gs.SessionTime, gs.DurationMinutes, gs.Notes, gs.CreatedAt, gs.CreatedBy)
}

func (r *careSessionRepo) ListGroupSessions(ctx context.Context, visitID uuid.UUID) ([]*caresession.GroupSession, error) {
	var rows []*caresession.GroupSession
	err := r.db.WithContext(ctx).
		Table("group_sessions gs").
		Joins("LEFT JOIN service_types st ON st.id = gs.service_type_id").
		Joins("LEFT JOIN amaur_workers w ON w.id = gs.worker_id").
		Select(`gs.*,
			st.name AS service_type_name,
			w.first_name AS worker_first_name, w.last_name AS worker_last_name`).
		Where("gs.visit_id = ?", visitID).
		Order("gs.session_date, gs.session_time").
		Scan(&rows).Error
	return rows, err
}

