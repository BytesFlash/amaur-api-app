package postgres

import (
	"context"

	"amaur/api/internal/domain/sessionrecord"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type sessionRecordRepo struct {
	db *gorm.DB
}

func NewSessionRecordRepository(db *gorm.DB) sessionrecord.Repository {
	return &sessionRecordRepo{db: db}
}

const sessionRecordSelectSQL = `
	SELECT
		sr.*,
		w.first_name || ' ' || w.last_name AS professional_name
	FROM session_records sr
	LEFT JOIN amaur_workers w ON w.id = sr.professional_id`

func (r *sessionRecordRepo) Create(ctx context.Context, sr *sessionrecord.SessionRecord) error {
	return rawExec(ctx, r.db, `
		INSERT INTO session_records (
			id, treatment_plan_id, appointment_id, patient_id, professional_id,
			session_number, evolution_notes, performed_treatment, patient_instructions,
			pain_level, next_action, follow_up_required, follow_up_date,
			internal_notes, created_by
		) VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,$9,
			$10,$11,$12,$13,
			$14,$15
		)`,
		sr.ID, sr.TreatmentPlanID, sr.AppointmentID, sr.PatientID, sr.ProfessionalID,
		sr.SessionNumber, sr.EvolutionNotes, sr.PerformedTreatment, sr.PatientInstructions,
		sr.PainLevel, sr.NextAction, sr.FollowUpRequired, sr.FollowUpDate,
		sr.InternalNotes, sr.CreatedBy)
}

func (r *sessionRecordRepo) FindByID(ctx context.Context, id uuid.UUID) (*sessionrecord.SessionRecord, error) {
	var sr sessionrecord.SessionRecord
	err := rawGet(ctx, r.db, &sr, sessionRecordSelectSQL+` WHERE sr.id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &sr, nil
}

func (r *sessionRecordRepo) Update(ctx context.Context, sr *sessionrecord.SessionRecord) error {
	return rawExec(ctx, r.db, `
		UPDATE session_records SET
			evolution_notes      = $1,
			performed_treatment  = $2,
			patient_instructions = $3,
			pain_level           = $4,
			next_action          = $5,
			follow_up_required   = $6,
			follow_up_date       = $7,
			internal_notes       = $8,
			updated_at           = NOW()
		WHERE id = $9`,
		sr.EvolutionNotes, sr.PerformedTreatment, sr.PatientInstructions,
		sr.PainLevel, sr.NextAction, sr.FollowUpRequired, sr.FollowUpDate,
		sr.InternalNotes, sr.ID)
}

func (r *sessionRecordRepo) ListByPlan(ctx context.Context, treatmentPlanID uuid.UUID) ([]*sessionrecord.SessionRecord, error) {
	var items []*sessionrecord.SessionRecord
	err := rawSelectPtr(ctx, r.db, &items,
		sessionRecordSelectSQL+` WHERE sr.treatment_plan_id = $1 ORDER BY sr.session_number ASC`,
		treatmentPlanID)
	return items, err
}

func (r *sessionRecordRepo) FindByAppointment(ctx context.Context, appointmentID uuid.UUID) (*sessionrecord.SessionRecord, error) {
	var sr sessionrecord.SessionRecord
	err := rawGet(ctx, r.db, &sr, sessionRecordSelectSQL+` WHERE sr.appointment_id = $1`, appointmentID)
	if err != nil {
		return nil, err
	}
	return &sr, nil
}
