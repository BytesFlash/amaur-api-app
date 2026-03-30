package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"amaur/api/internal/domain/caresession"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type careSessionRepo struct {
	db *sqlx.DB
}

func NewCareSessionRepository(db *sqlx.DB) caresession.Repository {
	return &careSessionRepo{db: db}
}

func (r *careSessionRepo) Create(ctx context.Context, cs *caresession.CareSession) error {
	cs.ID = uuid.New()
	cs.CreatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO care_sessions (
			id, visit_id, patient_id, worker_id, service_type_id, company_id,
			contract_service_id, session_type, session_date, session_time,
			duration_minutes, status, chief_complaint, subjective, objective,
			assessment, plan, notes, follow_up_required, created_at, created_by
		) VALUES (
			:id, :visit_id, :patient_id, :worker_id, :service_type_id, :company_id,
			:contract_service_id, :session_type, :session_date, :session_time,
			:duration_minutes, :status, :chief_complaint, :subjective, :objective,
			:assessment, :plan, :notes, :follow_up_required, :created_at, :created_by
		)`, cs)
	return err
}

func (r *careSessionRepo) FindByID(ctx context.Context, id uuid.UUID) (*caresession.CareSession, error) {
	var cs caresession.CareSession
	err := r.db.GetContext(ctx, &cs, `
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
	_, err := r.db.NamedExecContext(ctx, `
		UPDATE care_sessions SET
			status=:status, session_date=:session_date, session_time=:session_time,
			duration_minutes=:duration_minutes, chief_complaint=:chief_complaint,
			subjective=:subjective, objective=:objective, assessment=:assessment,
			plan=:plan, notes=:notes, follow_up_required=:follow_up_required,
			follow_up_status=:follow_up_status, follow_up_date=:follow_up_date,
			follow_up_notes=:follow_up_notes, updated_at=:updated_at, updated_by=:updated_by
		WHERE id=:id`, cs)
	return err
}

func (r *careSessionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM care_sessions WHERE id=$1`, id)
	return err
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

	var total int64
	if err := r.db.GetContext(ctx, &total, `
		SELECT COUNT(*) FROM care_sessions cs WHERE `+clause, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	var rows []*caresession.CareSession
	if err := r.db.SelectContext(ctx, &rows, fmt.Sprintf(`
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

	return rows, total, nil
}

func (r *careSessionRepo) CreateGroupSession(ctx context.Context, gs *caresession.GroupSession) error {
	gs.ID = uuid.New()
	gs.CreatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO group_sessions (
			id, visit_id, service_type_id, worker_id, attendee_count,
			session_date, session_time, duration_minutes, notes, created_at, created_by
		) VALUES (
			:id, :visit_id, :service_type_id, :worker_id, :attendee_count,
			:session_date, :session_time, :duration_minutes, :notes, :created_at, :created_by
		)`, gs)
	return err
}

func (r *careSessionRepo) ListGroupSessions(ctx context.Context, visitID uuid.UUID) ([]*caresession.GroupSession, error) {
	var rows []*caresession.GroupSession
	err := r.db.SelectContext(ctx, &rows, `
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
