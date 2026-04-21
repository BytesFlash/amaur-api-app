package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"amaur/api/internal/domain/program"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type programRepo struct {
	db *gorm.DB
}

func NewProgramRepository(db *gorm.DB) program.Repository {
	return &programRepo{db: db}
}

func (r *programRepo) CreateProgram(ctx context.Context, p *program.CompanyProgram) error {
	return rawExec(ctx, r.db, `
		INSERT INTO company_programs (
			id, company_id, contract_id, name, start_date, end_date,
			status, notes, created_by
		) VALUES (
			$1,$2,$3,$4,$5,$6,
			$7,$8,$9
		)
	`, p.ID, p.CompanyID, p.ContractID, p.Name, p.StartDate, p.EndDate, p.Status, p.Notes, p.CreatedBy)
}

func (r *programRepo) GetProgramByID(ctx context.Context, id uuid.UUID) (*program.CompanyProgram, error) {
	var p program.CompanyProgram
	err := rawGet(ctx, r.db, &p, `SELECT * FROM company_programs WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *programRepo) UpdateProgram(ctx context.Context, p *program.CompanyProgram) error {
	return rawExec(ctx, r.db, `
		UPDATE company_programs SET
			name = $1,
			start_date = $2,
			end_date = $3,
			status = $4,
			notes = $5,
			updated_at = NOW(),
			updated_by = $6
		WHERE id = $7
	`, p.Name, p.StartDate, p.EndDate, p.Status, p.Notes, p.UpdatedBy, p.ID)
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

	var totalRow struct {
		Count int64 `gorm:"column:count"`
	}
	if err := rawGet(ctx, r.db, &totalRow, `SELECT COUNT(*) AS count FROM company_programs WHERE `+whereClause, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows := []*program.CompanyProgram{}
	if err := rawSelectPtr(ctx, r.db, &rows,
		fmt.Sprintf(`SELECT * FROM company_programs WHERE %s ORDER BY start_date DESC LIMIT $%d OFFSET $%d`, whereClause, idx, idx+1),
		args...); err != nil {
		return nil, 0, err
	}

	return rows, totalRow.Count, nil
}

func (r *programRepo) CreateScheduleRules(ctx context.Context, rules []*program.ScheduleRule) error {
	for _, rule := range rules {
		if err := rawExec(ctx, r.db, `
			INSERT INTO company_program_schedule_rules (
				id, program_id, weekday, start_time, duration_minutes,
				frequency_interval_weeks, max_occurrences, service_type_id, worker_id, created_by
			) VALUES (
				$1,$2,$3,$4,$5,
				$6,$7,$8,$9,$10
			)
		`, rule.ID, rule.ProgramID, rule.Weekday, rule.StartTime, rule.DurationMinutes,
			rule.FrequencyIntervalWeeks, rule.MaxOccurrences, rule.ServiceTypeID, rule.WorkerID, rule.CreatedBy); err != nil {
			return err
		}
	}
	return nil
}

func (r *programRepo) ListScheduleRules(ctx context.Context, programID uuid.UUID) ([]*program.ScheduleRule, error) {
	rows := []*program.ScheduleRule{}
	err := rawSelectPtr(ctx, r.db, &rows, `
		SELECT
			id,
			program_id,
			weekday,
			TO_CHAR(start_time, 'HH24:MI') AS start_time,
			duration_minutes,
			frequency_interval_weeks,
			max_occurrences,
			service_type_id,
			worker_id,
			created_at,
			created_by
		FROM company_program_schedule_rules
		WHERE program_id = $1
		ORDER BY weekday, start_time
	`, programID)
	return rows, err
}

func (r *programRepo) ReplaceScheduleRules(ctx context.Context, programID uuid.UUID, rules []*program.ScheduleRule) error {
	return withTx(ctx, r.db, func(tx *gorm.DB) error {
		if err := rawExec(ctx, tx, `DELETE FROM company_program_schedule_rules WHERE program_id = $1`, programID); err != nil {
			return err
		}
		for _, rule := range rules {
			if err := rawExec(ctx, tx, `
				INSERT INTO company_program_schedule_rules (
					id, program_id, weekday, start_time, duration_minutes,
					frequency_interval_weeks, max_occurrences, service_type_id, worker_id, created_by
				) VALUES (
					$1,$2,$3,$4,$5,
					$6,$7,$8,$9,$10
				)
			`, rule.ID, rule.ProgramID, rule.Weekday, rule.StartTime, rule.DurationMinutes,
				rule.FrequencyIntervalWeeks, rule.MaxOccurrences, rule.ServiceTypeID, rule.WorkerID, rule.CreatedBy); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *programRepo) CreateAgendaServices(ctx context.Context, services []*program.AgendaService) error {
	for _, service := range services {
		if err := rawExec(ctx, r.db, `
			INSERT INTO agenda_services (
				id, agenda_id, service_type_id, worker_id, planned_start_time,
				planned_duration_minutes, status, notes
			) VALUES (
				$1,$2,$3,$4,$5,
				$6,$7,$8
			)
		`, service.ID, service.AgendaID, service.ServiceTypeID, service.WorkerID, service.PlannedStartTime,
			service.PlannedDurationMinutes, service.Status, service.Notes); err != nil {
			return err
		}
	}
	return nil
}

func (r *programRepo) ListAgendaServices(ctx context.Context, agendaID uuid.UUID) ([]*program.AgendaService, error) {
	rows := []*program.AgendaService{}
	err := rawSelectPtr(ctx, r.db, &rows, `
		SELECT
			id,
			agenda_id,
			service_type_id,
			worker_id,
			TO_CHAR(planned_start_time, 'HH24:MI') AS planned_start_time,
			planned_duration_minutes,
			status,
			notes,
			completed_at,
			completed_by,
			created_at,
			updated_at
		FROM agenda_services
		WHERE agenda_id = $1
		ORDER BY planned_start_time NULLS LAST, created_at ASC
	`, agendaID)
	return rows, err
}

func (r *programRepo) GetAgendaServiceByID(ctx context.Context, id uuid.UUID) (*program.AgendaService, error) {
	var row program.AgendaService
	err := rawGet(ctx, r.db, &row, `SELECT * FROM agenda_services WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *programRepo) GetAgendaContextByServiceID(ctx context.Context, agendaServiceID uuid.UUID) (*program.AgendaServiceContext, error) {
	var row program.AgendaServiceContext
	err := rawGet(ctx, r.db, &row, `
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

func (r *programRepo) GetAgendaContextByAgendaID(ctx context.Context, agendaID uuid.UUID) (*program.AgendaServiceContext, error) {
	var row program.AgendaServiceContext
	err := rawGet(ctx, r.db, &row, `
		SELECT id AS agenda_id, company_id, scheduled_date, TO_CHAR(scheduled_start, 'HH24:MI') AS scheduled_start
		FROM agendas
		WHERE id = $1
	`, agendaID)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *programRepo) UpdateAgendaService(ctx context.Context, service *program.AgendaService) error {
	return rawExec(ctx, r.db, `
		UPDATE agenda_services SET
			worker_id = $1,
			planned_start_time = $2,
			planned_duration_minutes = $3,
			status = $4,
			notes = $5,
			completed_at = $6,
			completed_by = $7,
			updated_at = NOW()
		WHERE id = $8
	`, service.WorkerID, service.PlannedStartTime, service.PlannedDurationMinutes, service.Status, service.Notes, service.CompletedAt, service.CompletedBy, service.ID)
}

func (r *programRepo) UpsertParticipants(ctx context.Context, participants []*program.AgendaServiceParticipant) error {
	for _, p := range participants {
		if p.Attended && p.AttendedAt == nil {
			now := time.Now()
			p.AttendedAt = &now
		}
		if err := rawExec(ctx, r.db, `
			INSERT INTO agenda_service_participants (
				id, agenda_service_id, patient_id, attended, attended_at, notes, created_by
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7
			)
			ON CONFLICT (agenda_service_id, patient_id) DO UPDATE SET
				attended = EXCLUDED.attended,
				attended_at = EXCLUDED.attended_at,
				notes = EXCLUDED.notes
		`, p.ID, p.AgendaServiceID, p.PatientID, p.Attended, p.AttendedAt, p.Notes, p.CreatedBy); err != nil {
			return err
		}
	}
	return nil
}

func (r *programRepo) ListParticipants(ctx context.Context, agendaServiceID uuid.UUID) ([]*program.AgendaServiceParticipant, error) {
	rows := []*program.AgendaServiceParticipant{}
	err := rawSelectPtr(ctx, r.db, &rows, `
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

	var ctxRow struct {
		CompanyID uuid.UUID `gorm:"column:company_id"`
	}
	if err := rawGet(ctx, r.db, &ctxRow, `
		SELECT a.company_id
		FROM agenda_services s
		JOIN agendas a ON a.id = s.agenda_id
		WHERE s.id = $1
	`, agendaServiceID); err != nil {
		if err == sql.ErrNoRows {
			return patientIDs, nil
		}
		return nil, err
	}

	type row struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).Raw(`
		SELECT p.id
		FROM patients p
		WHERE p.id IN ?
		AND EXISTS (
			SELECT 1
			FROM patient_companies pc
			WHERE pc.patient_id = p.id
			AND pc.company_id = ?
			AND pc.is_active = true
		)
	`, patientIDs, ctxRow.CompanyID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	validSet := make(map[uuid.UUID]struct{}, len(rows))
	for _, row := range rows {
		validSet[row.ID] = struct{}{}
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
	return rawExec(ctx, r.db, `
		INSERT INTO company_program_agendas (program_id, agenda_id)
		VALUES ($1, $2)
		ON CONFLICT (program_id, agenda_id) DO NOTHING
	`, programID, agendaID)
}

func (r *programRepo) ListParticipantsDetail(ctx context.Context, agendaServiceID uuid.UUID) ([]*program.ParticipantDetail, error) {
	rows := []*program.ParticipantDetail{}
	err := rawSelectPtr(ctx, r.db, &rows, `
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

func (r *programRepo) ListProgramAgendas(ctx context.Context, programID uuid.UUID) ([]*program.AgendaWithServices, error) {
	type agendaRow struct {
		AgendaID       uuid.UUID `gorm:"column:agenda_id"`
		ScheduledDate  time.Time `gorm:"column:scheduled_date"`
		ScheduledStart *string   `gorm:"column:scheduled_start"`
		Status         string    `gorm:"column:status"`
	}
	var aRows []agendaRow
	if err := rawSelect(ctx, r.db, &aRows, `
		SELECT
			a.id AS agenda_id,
			a.scheduled_date,
			TO_CHAR(a.scheduled_start, 'HH24:MI') AS scheduled_start,
			a.status
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

	type svcRow struct {
		program.AgendaService
		ServiceTypeName *string `gorm:"column:service_type_name"`
		WorkerName      *string `gorm:"column:worker_name"`
	}
	var svcRows []svcRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			s.id,
			s.agenda_id,
			s.service_type_id,
			s.worker_id,
			TO_CHAR(s.planned_start_time, 'HH24:MI') AS planned_start_time,
			s.planned_duration_minutes,
			s.status,
			s.notes,
			s.completed_at,
			s.completed_by,
			s.created_at,
			s.updated_at,
			st.name AS service_type_name,
			w.first_name || ' ' || w.last_name AS worker_name
		FROM agenda_services s
		JOIN service_types st ON st.id = s.service_type_id
		LEFT JOIN amaur_workers w ON w.id = s.worker_id
		WHERE s.agenda_id IN ?
		ORDER BY s.agenda_id, s.planned_start_time NULLS LAST, s.created_at
	`, agendaIDs).Scan(&svcRows).Error; err != nil {
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
	err := rawExec(ctx, r.db, `
		INSERT INTO agendas (id, company_id, contract_id, status, scheduled_date, scheduled_start, created_by)
		VALUES ($1, $2, $3, 'scheduled', $4, $5, $6)
	`, id, companyID, contractID, scheduledDate, scheduledStart, byUserID)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *programRepo) ListCompanyPatientIDs(ctx context.Context, companyID uuid.UUID) ([]uuid.UUID, error) {
	type row struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	var rows []row
	err := rawSelect(ctx, r.db, &rows, `
		SELECT p.id
		FROM patient_companies pc
		JOIN patients p ON p.id = pc.patient_id
		WHERE pc.company_id = $1
		  AND pc.is_active = true
		  AND p.deleted_at IS NULL
		ORDER BY p.first_name, p.last_name
	`, companyID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids, nil
}

func (r *programRepo) HasWorkerScheduleConflict(ctx context.Context, workerID uuid.UUID, scheduledDate time.Time, startTime string, durationMinutes int, excludeAgendaServiceID *uuid.UUID) (bool, error) {
	if durationMinutes <= 0 {
		durationMinutes = 60
	}

	normalized := normalizeClockString(startTime)
	start, err := time.Parse("15:04", normalized)
	if err != nil {
		return false, err
	}
	scheduledStart := time.Date(
		scheduledDate.Year(),
		scheduledDate.Month(),
		scheduledDate.Day(),
		start.Hour(),
		start.Minute(),
		0,
		0,
		time.UTC,
	)

	var excluded sql.NullString
	if excludeAgendaServiceID != nil {
		excluded.Valid = true
		excluded.String = excludeAgendaServiceID.String()
	}

	query := `
		SELECT EXISTS (
			SELECT 1
			FROM agenda_services s
			JOIN agendas a ON a.id = s.agenda_id
			WHERE s.worker_id = $1
			  AND s.status IN ('planned', 'completed')
			  AND a.scheduled_date = $2::date
			  AND (a.scheduled_date + COALESCE(s.planned_start_time, a.scheduled_start)) < ($3::timestamp + ($4::int * INTERVAL '1 minute'))
			  AND ((a.scheduled_date + COALESCE(s.planned_start_time, a.scheduled_start)) + (COALESCE(s.planned_duration_minutes, 60) * INTERVAL '1 minute')) > $3::timestamp
			  AND ($5::uuid IS NULL OR s.id <> $5)
		) AS exists
	`

	var row struct {
		Exists bool `gorm:"column:exists"`
	}
	if err := rawGet(ctx, r.db, &row, query, workerID, scheduledDate, scheduledStart, durationMinutes, excluded); err != nil {
		return false, err
	}
	return row.Exists, nil
}

func normalizeClockString(raw string) string {
	if raw == "" {
		return raw
	}
	if t, err := time.Parse("15:04:05", raw); err == nil {
		return t.Format("15:04")
	}
	if t, err := time.Parse("15:04", raw); err == nil {
		return t.Format("15:04")
	}
	return raw
}

func (r *programRepo) ListAgendaServicesByWorker(ctx context.Context, workerID uuid.UUID, from, to time.Time) ([]*program.AgendaServiceWithDate, error) {
	rows := []*program.AgendaServiceWithDate{}
	err := rawSelectPtr(ctx, r.db, &rows, `
		SELECT
			s.id,
			s.agenda_id,
			s.service_type_id,
			s.worker_id,
			TO_CHAR(s.planned_start_time, 'HH24:MI') AS planned_start_time,
			s.planned_duration_minutes,
			s.status,
			s.notes,
			s.completed_at,
			s.completed_by,
			s.created_at,
			s.updated_at,
			a.scheduled_date,
			st.name AS service_type_name
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
