package postgres

import (
	"context"
	"fmt"
	"strings"
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
	err := rawGet(ctx, r.db, &cs, `
		SELECT cs.*,
			p.first_name AS patient_first_name, p.last_name AS patient_last_name,
			w.first_name AS worker_first_name, w.last_name AS worker_last_name,
			st.name AS service_type_name,
			c.name AS company_name
		FROM care_sessions cs
		JOIN patients p ON p.id = cs.patient_id
		JOIN amaur_workers w ON w.id = cs.worker_id
		JOIN service_types st ON st.id = cs.service_type_id
		LEFT JOIN companies c ON c.id = cs.company_id
		WHERE cs.id = $1`, id)
	if err != nil {
		return nil, err
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
	where := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if f.PatientID != nil {
		where = append(where, fmt.Sprintf("cs.patient_id=$%d", idx))
		args = append(args, *f.PatientID)
		idx++
	}
	if f.WorkerID != nil {
		where = append(where, fmt.Sprintf("cs.worker_id=$%d", idx))
		args = append(args, *f.WorkerID)
		idx++
	}
	if f.CompanyID != nil {
		where = append(where, fmt.Sprintf("cs.company_id=$%d", idx))
		args = append(args, *f.CompanyID)
		idx++
	}
	if f.VisitID != nil {
		where = append(where, fmt.Sprintf("cs.visit_id=$%d", idx))
		args = append(args, *f.VisitID)
		idx++
	}
	if f.SessionType != "" {
		where = append(where, fmt.Sprintf("cs.session_type=$%d", idx))
		args = append(args, f.SessionType)
		idx++
	}
	if f.Status != "" {
		where = append(where, fmt.Sprintf("cs.status=$%d", idx))
		args = append(args, f.Status)
		idx++
	}
	if f.DateFrom != nil {
		where = append(where, fmt.Sprintf("cs.session_date>=$%d", idx))
		args = append(args, *f.DateFrom)
		idx++
	}
	if f.DateTo != nil {
		where = append(where, fmt.Sprintf("cs.session_date<=$%d", idx))
		args = append(args, *f.DateTo)
		idx++
	}

	clause := strings.Join(where, " AND ")

	var totalRow struct {
		Count int64 `gorm:"column:count"`
	}
	if err := rawGet(ctx, r.db, &totalRow, `SELECT COUNT(*) AS count FROM care_sessions cs WHERE `+clause, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	var rows []*caresession.CareSession
	if err := rawSelectPtr(ctx, r.db, &rows, fmt.Sprintf(`
		SELECT cs.*,
			p.first_name AS patient_first_name, p.last_name AS patient_last_name,
			w.first_name AS worker_first_name, w.last_name AS worker_last_name,
			st.name AS service_type_name,
			c.name AS company_name
		FROM care_sessions cs
		JOIN patients p ON p.id = cs.patient_id
		JOIN amaur_workers w ON w.id = cs.worker_id
		JOIN service_types st ON st.id = cs.service_type_id
		LEFT JOIN companies c ON c.id = cs.company_id
		WHERE %s
		ORDER BY cs.session_date DESC, cs.session_time DESC
		LIMIT $%d OFFSET $%d`, clause, idx, idx+1), args...); err != nil {
		return nil, 0, err
	}

	return rows, totalRow.Count, nil
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
	err := rawSelectPtr(ctx, r.db, &rows, `
		SELECT gs.*,
			st.name AS service_type_name,
			w.first_name AS worker_first_name, w.last_name AS worker_last_name
		FROM group_sessions gs
		JOIN service_types st ON st.id = gs.service_type_id
		LEFT JOIN amaur_workers w ON w.id = gs.worker_id
		WHERE gs.visit_id = $1
		ORDER BY gs.session_date, gs.session_time`, visitID)
	return rows, err
}
