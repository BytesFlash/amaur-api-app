package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"amaur/api/internal/domain/program"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type programRepo struct {
	db *sqlx.DB
}

func NewProgramRepository(db *sqlx.DB) program.Repository {
	return &programRepo{db: db}
}

func (r *programRepo) CreateProgram(ctx context.Context, p *program.CompanyProgram) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO company_programs (
			id, company_id, contract_id, name, start_date, end_date,
			status, notes, created_by
		) VALUES (
			:id, :company_id, :contract_id, :name, :start_date, :end_date,
			:status, :notes, :created_by
		)
	`, p)
	return err
}

func (r *programRepo) GetProgramByID(ctx context.Context, id uuid.UUID) (*program.CompanyProgram, error) {
	var p program.CompanyProgram
	err := r.db.GetContext(ctx, &p, `SELECT * FROM company_programs WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *programRepo) UpdateProgram(ctx context.Context, p *program.CompanyProgram) error {
	_, err := r.db.NamedExecContext(ctx, `
		UPDATE company_programs SET
			name = :name,
			start_date = :start_date,
			end_date = :end_date,
			status = :status,
			notes = :notes,
			updated_at = NOW(),
			updated_by = :updated_by
		WHERE id = :id
	`, p)
	return err
}

func (r *programRepo) ListPrograms(ctx context.Context, f program.Filter, limit, offset int) ([]*program.CompanyProgram, int64, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if f.CompanyID != nil {
		where = append(where, fmt.Sprintf("company_id = $%d", idx))
		args = append(args, *f.CompanyID)
		idx++
	}
	if f.ContractID != nil {
		where = append(where, fmt.Sprintf("contract_id = $%d", idx))
		args = append(args, *f.ContractID)
		idx++
	}
	if f.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", idx))
		args = append(args, f.Status)
		idx++
	}
	if f.DateFrom != nil {
		where = append(where, fmt.Sprintf("start_date >= $%d", idx))
		args = append(args, *f.DateFrom)
		idx++
	}
	if f.DateTo != nil {
		where = append(where, fmt.Sprintf("COALESCE(end_date, start_date) <= $%d", idx))
		args = append(args, *f.DateTo)
		idx++
	}

	whereClause := strings.Join(where, " AND ")

	var total int64
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM company_programs WHERE `+whereClause, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows := []*program.CompanyProgram{}
	if err := r.db.SelectContext(ctx, &rows,
		fmt.Sprintf(`SELECT * FROM company_programs WHERE %s ORDER BY start_date DESC LIMIT $%d OFFSET $%d`, whereClause, idx, idx+1),
		args...,
	); err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *programRepo) CreateScheduleRules(ctx context.Context, rules []*program.ScheduleRule) error {
	for _, rule := range rules {
		_, err := r.db.NamedExecContext(ctx, `
			INSERT INTO company_program_schedule_rules (
				id, program_id, weekday, start_time, duration_minutes,
				frequency_interval_weeks, max_occurrences, service_type_id, worker_id, created_by
			) VALUES (
				:id, :program_id, :weekday, :start_time, :duration_minutes,
				:frequency_interval_weeks, :max_occurrences, :service_type_id, :worker_id, :created_by
			)
		`, rule)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *programRepo) ListScheduleRules(ctx context.Context, programID uuid.UUID) ([]*program.ScheduleRule, error) {
	rows := []*program.ScheduleRule{}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT * FROM company_program_schedule_rules
		WHERE program_id = $1
		ORDER BY weekday, start_time
	`, programID)
	return rows, err
}

func (r *programRepo) ReplaceScheduleRules(ctx context.Context, programID uuid.UUID, rules []*program.ScheduleRule) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM company_program_schedule_rules WHERE program_id = $1`, programID); err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, rule := range rules {
		_, err := tx.NamedExecContext(ctx, `
			INSERT INTO company_program_schedule_rules (
				id, program_id, weekday, start_time, duration_minutes,
				frequency_interval_weeks, max_occurrences, service_type_id, worker_id, created_by
			) VALUES (
				:id, :program_id, :weekday, :start_time, :duration_minutes,
				:frequency_interval_weeks, :max_occurrences, :service_type_id, :worker_id, :created_by
			)
		`, rule)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (r *programRepo) CreateAgendaServices(ctx context.Context, services []*program.AgendaService) error {
	for _, service := range services {
		_, err := r.db.NamedExecContext(ctx, `
			INSERT INTO agenda_services (
				id, agenda_id, service_type_id, worker_id, planned_start_time,
				planned_duration_minutes, status, notes
			) VALUES (
				:id, :agenda_id, :service_type_id, :worker_id, :planned_start_time,
				:planned_duration_minutes, :status, :notes
			)
		`, service)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *programRepo) ListAgendaServices(ctx context.Context, agendaID uuid.UUID) ([]*program.AgendaService, error) {
	rows := []*program.AgendaService{}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT * FROM agenda_services
		WHERE agenda_id = $1
		ORDER BY planned_start_time NULLS LAST, created_at ASC
	`, agendaID)
	return rows, err
}

func (r *programRepo) GetAgendaServiceByID(ctx context.Context, id uuid.UUID) (*program.AgendaService, error) {
	var row program.AgendaService
	err := r.db.GetContext(ctx, &row, `SELECT * FROM agenda_services WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *programRepo) GetAgendaContextByServiceID(ctx context.Context, agendaServiceID uuid.UUID) (*program.AgendaServiceContext, error) {
	var row program.AgendaServiceContext
	err := r.db.GetContext(ctx, &row, `
		SELECT a.id AS agenda_id, a.company_id, a.scheduled_date, a.scheduled_start
		FROM agenda_services s
		JOIN agendas a ON a.id = s.agenda_id
		WHERE s.id = $1
	`, agendaServiceID)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *programRepo) UpdateAgendaService(ctx context.Context, service *program.AgendaService) error {
	_, err := r.db.NamedExecContext(ctx, `
		UPDATE agenda_services SET
			worker_id = :worker_id,
			planned_start_time = :planned_start_time,
			planned_duration_minutes = :planned_duration_minutes,
			status = :status,
			notes = :notes,
			completed_at = :completed_at,
			completed_by = :completed_by,
			updated_at = NOW()
		WHERE id = :id
	`, service)
	return err
}

func (r *programRepo) UpsertParticipants(ctx context.Context, participants []*program.AgendaServiceParticipant) error {
	for _, p := range participants {
		if p.Attended && p.AttendedAt == nil {
			now := time.Now()
			p.AttendedAt = &now
		}
		_, err := r.db.NamedExecContext(ctx, `
			INSERT INTO agenda_service_participants (
				id, agenda_service_id, patient_id, attended, attended_at, notes, created_by
			) VALUES (
				:id, :agenda_service_id, :patient_id, :attended, :attended_at, :notes, :created_by
			)
			ON CONFLICT (agenda_service_id, patient_id) DO UPDATE SET
				attended = EXCLUDED.attended,
				attended_at = EXCLUDED.attended_at,
				notes = EXCLUDED.notes
		`, p)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *programRepo) ListParticipants(ctx context.Context, agendaServiceID uuid.UUID) ([]*program.AgendaServiceParticipant, error) {
	rows := []*program.AgendaServiceParticipant{}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT * FROM agenda_service_participants
		WHERE agenda_service_id = $1
		ORDER BY created_at ASC
	`, agendaServiceID)
	return rows, err
}

func (r *programRepo) PatientIDsOutsideAgendaCompany(ctx context.Context, agendaServiceID uuid.UUID, patientIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(patientIDs) == 0 {
		return nil, nil
	}

	ctxRow := r.db.QueryRowxContext(ctx, `
		SELECT a.company_id
		FROM agenda_services s
		JOIN agendas a ON a.id = s.agenda_id
		WHERE s.id = $1
	`, agendaServiceID)

	var companyID uuid.UUID
	if err := ctxRow.Scan(&companyID); err != nil {
		if err == sql.ErrNoRows {
			return patientIDs, nil
		}
		return nil, err
	}

	args := []interface{}{companyID}
	placeholders := make([]string, 0, len(patientIDs))
	for i, id := range patientIDs {
		args = append(args, id)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+2))
	}

	query := `
		SELECT p.id
		FROM patients p
		WHERE p.id IN (` + strings.Join(placeholders, ",") + `)
		AND EXISTS (
			SELECT 1
			FROM patient_companies pc
			WHERE pc.patient_id = p.id
			AND pc.company_id = $1
			AND pc.is_active = true
		)
	`

	var valid []uuid.UUID
	if err := r.db.SelectContext(ctx, &valid, query, args...); err != nil {
		return nil, err
	}

	validSet := make(map[uuid.UUID]struct{}, len(valid))
	for _, id := range valid {
		validSet[id] = struct{}{}
	}

	missing := make([]uuid.UUID, 0)
	for _, id := range patientIDs {
		if _, ok := validSet[id]; !ok {
			missing = append(missing, id)
		}
	}

	return missing, nil
}

func (r *programRepo) LinkProgramAgenda(ctx context.Context, programID, agendaID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO company_program_agendas (program_id, agenda_id)
		VALUES ($1, $2)
		ON CONFLICT (program_id, agenda_id) DO NOTHING
	`, programID, agendaID)
	return err
}

func (r *programRepo) ListParticipantsDetail(ctx context.Context, agendaServiceID uuid.UUID) ([]*program.ParticipantDetail, error) {
	rows := []*program.ParticipantDetail{}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT
			asp.*,
			p.first_name || ' ' || p.last_name AS patient_name
		FROM agenda_service_participants asp
		JOIN patients p ON p.id = asp.patient_id
		WHERE asp.agenda_service_id = $1
		ORDER BY patient_name ASC
	`, agendaServiceID)
	return rows, err
}

func (r *programRepo) ListProgramAgendas(ctx context.Context, programID uuid.UUID) ([]*program.AgendaWithServices, error) { // 1. Get agendas linked to the program
	type agendaRow struct {
		AgendaID       uuid.UUID `db:"agenda_id"`
		ScheduledDate  time.Time `db:"scheduled_date"`
		ScheduledStart *string   `db:"scheduled_start"`
		Status         string    `db:"status"`
	}
	var aRows []agendaRow
	if err := r.db.SelectContext(ctx, &aRows, `
		SELECT a.id AS agenda_id, a.scheduled_date, a.scheduled_start, a.status
		FROM agendas a
		JOIN company_program_agendas cpa ON cpa.agenda_id = a.id
		WHERE cpa.program_id = $1
		ORDER BY a.scheduled_date ASC, a.scheduled_start ASC
	`, programID); err != nil {
		return nil, err
	}

	result := make([]*program.AgendaWithServices, 0, len(aRows))
	agendaIDs := make([]uuid.UUID, 0, len(aRows))
	agendaIndex := make(map[uuid.UUID]int, len(aRows))

	for i, ar := range aRows {
		result = append(result, &program.AgendaWithServices{
			AgendaID:       ar.AgendaID,
			ScheduledDate:  ar.ScheduledDate,
			ScheduledStart: ar.ScheduledStart,
			Status:         ar.Status,
			Services:       []*program.AgendaServiceDetail{},
		})
		agendaIDs = append(agendaIDs, ar.AgendaID)
		agendaIndex[ar.AgendaID] = i
	}

	if len(agendaIDs) == 0 {
		return result, nil
	}

	// 2. Load services for all those agendas in one query
	placeholders := make([]string, len(agendaIDs))
	args := make([]interface{}, len(agendaIDs))
	for i, id := range agendaIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	type svcRow struct {
		program.AgendaService
		ServiceTypeName *string `db:"service_type_name"`
		WorkerName      *string `db:"worker_name"`
	}
	var svcRows []svcRow
	if err := r.db.SelectContext(ctx, &svcRows, `
		SELECT
			s.*,
			st.name AS service_type_name,
			w.first_name || ' ' || w.last_name AS worker_name
		FROM agenda_services s
		JOIN service_types st ON st.id = s.service_type_id
		LEFT JOIN amaur_workers w ON w.id = s.worker_id
		WHERE s.agenda_id IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY s.agenda_id, s.planned_start_time NULLS LAST, s.created_at
	`, args...); err != nil {
		return nil, err
	}

	for i := range svcRows {
		sr := &svcRows[i]
		detail := &program.AgendaServiceDetail{
			AgendaService:   sr.AgendaService,
			ServiceTypeName: sr.ServiceTypeName,
			WorkerName:      sr.WorkerName,
		}
		if idx, ok := agendaIndex[sr.AgendaID]; ok {
			result[idx].Services = append(result[idx].Services, detail)
		}
	}

	return result, nil
}

func (r *programRepo) CreateAgenda(ctx context.Context, companyID uuid.UUID, contractID *uuid.UUID, scheduledDate time.Time, scheduledStart *string, byUserID uuid.UUID) (uuid.UUID, error) {
	id := uuid.New()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO agendas (id, company_id, contract_id, status, scheduled_date, scheduled_start, created_by)
		VALUES ($1, $2, $3, 'scheduled', $4, $5, $6)
	`, id, companyID, contractID, scheduledDate, scheduledStart, byUserID)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// ListAgendaServicesByWorker returns agenda_services assigned to a worker where the
// agenda's scheduled_date falls within [from, to), including the date on the service.
func (r *programRepo) ListAgendaServicesByWorker(ctx context.Context, workerID uuid.UUID, from, to time.Time) ([]*program.AgendaServiceWithDate, error) {
	rows := []*program.AgendaServiceWithDate{}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT s.*, a.scheduled_date, st.name AS service_type_name
		FROM agenda_services s
		JOIN agendas a ON a.id = s.agenda_id
		JOIN service_types st ON st.id = s.service_type_id
		WHERE s.worker_id = $1
		  AND s.status IN ('planned', 'completed')
		  AND a.scheduled_date >= $2
		  AND a.scheduled_date < $3
		ORDER BY a.scheduled_date, s.planned_start_time NULLS LAST
	`, workerID, from, to)
	return rows, err
}
