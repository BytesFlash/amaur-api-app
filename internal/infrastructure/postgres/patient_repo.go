package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"amaur/api/internal/domain/patient"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// dbx is satisfied by both *sqlx.DB and *sqlx.Tx, enabling the same
// repository code to run inside or outside a transaction.
type dbx interface {
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

type PatientRepository struct {
	db   *sqlx.DB // used only to begin transactions
	q    dbx      // active executor (db or tx)
	inTx bool     // guards against nested InTx calls
}

func NewPatientRepository(db *sqlx.DB) *PatientRepository {
	return &PatientRepository{db: db, q: db}
}

func (r *PatientRepository) withTx(tx *sqlx.Tx) *PatientRepository {
	return &PatientRepository{db: r.db, q: tx, inTx: true}
}

// InTx executes fn inside a single database transaction.
// If called on a repository that is already inside a transaction the
// existing transaction is reused transparently.
func (r *PatientRepository) InTx(ctx context.Context, fn func(patient.Repository) error) error {
	if r.inTx {
		return fn(r)
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(r.withTx(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// ── has_login subquery ───────────────────────────────────────────────────────

const hasLoginExpr = `EXISTS(
	SELECT 1 FROM users u
	WHERE u.patient_id = p.id AND u.deleted_at IS NULL
) AS has_login`

// ── Core CRUD ────────────────────────────────────────────────────────────────

func (r *PatientRepository) Create(ctx context.Context, p *patient.Patient) error {
	_, err := r.q.ExecContext(ctx,
		`INSERT INTO patients
         (id, rut, first_name, last_name, birth_date, gender, email, phone, address, city, region,
          emergency_contact_name, emergency_contact_phone, general_notes,
          patient_type, status, tutor_id, created_at, created_by)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		p.ID, p.RUT, p.FirstName, p.LastName, p.BirthDate, p.Gender, p.Email, p.Phone,
		p.Address, p.City, p.Region, p.EmergencyContactName, p.EmergencyContactPhone,
		p.GeneralNotes, p.PatientType, p.Status, p.TutorID, p.CreatedAt, p.CreatedBy)
	return err
}

func (r *PatientRepository) FindByID(ctx context.Context, id uuid.UUID) (*patient.Patient, error) {
	p := &patient.Patient{}
	err := r.q.GetContext(ctx, p,
		`SELECT p.id, p.rut, p.first_name, p.last_name, p.birth_date, p.gender, p.email,
                p.phone, p.address, p.city, p.region,
                p.emergency_contact_name, p.emergency_contact_phone, p.general_notes,
                p.patient_type, p.status, p.tutor_id,
                p.created_at, p.updated_at, p.created_by, p.updated_by,
                `+hasLoginExpr+`
         FROM patients p
         WHERE p.id = $1 AND p.deleted_at IS NULL`, id)
	return p, err
}

func (r *PatientRepository) FindByRUT(ctx context.Context, rut string) (*patient.Patient, error) {
	p := &patient.Patient{}
	err := r.q.GetContext(ctx, p,
		`SELECT p.id, p.rut, p.first_name, p.last_name, p.status, p.patient_type,
                p.tutor_id, p.created_at,
                `+hasLoginExpr+`
         FROM patients p
         WHERE p.rut = $1 AND p.deleted_at IS NULL`, rut)
	return p, err
}

func (r *PatientRepository) Update(ctx context.Context, p *patient.Patient) error {
	_, err := r.q.ExecContext(ctx,
		`UPDATE patients SET
         rut=$1, first_name=$2, last_name=$3, birth_date=$4, gender=$5, email=$6, phone=$7,
         address=$8, city=$9, region=$10, emergency_contact_name=$11, emergency_contact_phone=$12,
         general_notes=$13, patient_type=$14, status=$15, tutor_id=$16,
         updated_at=$17, updated_by=$18
         WHERE id=$19 AND deleted_at IS NULL`,
		p.RUT, p.FirstName, p.LastName, p.BirthDate, p.Gender, p.Email, p.Phone,
		p.Address, p.City, p.Region, p.EmergencyContactName, p.EmergencyContactPhone,
		p.GeneralNotes, p.PatientType, p.Status, p.TutorID,
		p.UpdatedAt, p.UpdatedBy, p.ID)
	return err
}

func (r *PatientRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.q.ExecContext(ctx,
		`UPDATE patients SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *PatientRepository) List(ctx context.Context, f patient.Filter, limit, offset int) ([]*patient.Patient, int64, error) {
	where := []string{"p.deleted_at IS NULL"}
	args := []interface{}{}
	argN := 1

	if f.Search != "" {
		where = append(where, fmt.Sprintf(
			`(p.first_name || ' ' || p.last_name ILIKE $%d OR p.rut ILIKE $%d OR p.email ILIKE $%d)`,
			argN, argN+1, argN+2,
		))
		q := "%" + f.Search + "%"
		args = append(args, q, q, q)
		argN += 3
	}
	if f.Status != "" {
		where = append(where, fmt.Sprintf("p.status = $%d", argN))
		args = append(args, f.Status)
		argN++
	}
	if f.PatientType != "" {
		where = append(where, fmt.Sprintf("p.patient_type = $%d", argN))
		args = append(args, f.PatientType)
		argN++
	}
	if f.CompanyID != nil {
		where = append(where, fmt.Sprintf(
			`EXISTS (SELECT 1 FROM patient_companies pc WHERE pc.patient_id=p.id AND pc.company_id=$%d AND pc.is_active=true)`,
			argN,
		))
		args = append(args, *f.CompanyID)
		argN++
	}
	if f.FollowUpPending {
		where = append(where, fmt.Sprintf(
			`EXISTS (SELECT 1 FROM care_sessions cs WHERE cs.patient_id=p.id AND cs.follow_up_required=true AND cs.follow_up_date <= $%d)`,
			argN,
		))
		args = append(args, time.Now().AddDate(0, 0, 7).Format("2006-01-02"))
		argN++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	query := fmt.Sprintf(
		`SELECT p.id, p.rut, p.first_name, p.last_name, p.email, p.phone,
                p.patient_type, p.status, p.tutor_id, p.created_at, p.updated_at,
                `+hasLoginExpr+`
         FROM patients p %s
         ORDER BY p.created_at DESC
         LIMIT $%d OFFSET $%d`,
		whereClause, argN, argN+1,
	)
	args = append(args, limit, offset)

	var items []*patient.Patient
	if err := r.q.SelectContext(ctx, &items, query, args...); err != nil {
		return nil, 0, err
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM patients p %s`, whereClause)
	var total int64
	_ = r.q.GetContext(ctx, &total, countQuery, args[:len(args)-2]...)

	return items, total, nil
}

// ── Company associations — FULL REPLACE ──────────────────────────────────────

// ReplaceCompanies atomically replaces all company associations for a patient.
// Delete + Insert run against the same executor (db or tx), so the caller
// MUST call this inside InTx when other writes are also in flight.
func (r *PatientRepository) ReplaceCompanies(ctx context.Context, patientID uuid.UUID, companies []*patient.PatientCompany) error {
	_, err := r.q.ExecContext(ctx,
		`DELETE FROM patient_companies WHERE patient_id = $1`, patientID)
	if err != nil {
		return err
	}
	for _, pc := range companies {
		_, err := r.q.ExecContext(ctx,
			`INSERT INTO patient_companies
             (id, patient_id, company_id, position, department, is_active, start_date, notes, created_at, created_by)
             VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			pc.ID, pc.PatientID, pc.CompanyID, pc.Position, pc.Department, pc.IsActive,
			pc.StartDate, pc.Notes, pc.CreatedAt, pc.CreatedBy)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *PatientRepository) ListPatientCompanies(ctx context.Context, patientID uuid.UUID) ([]*patient.PatientCompany, error) {
	var items []*patient.PatientCompany
	err := r.q.SelectContext(ctx, &items,
		`SELECT id, patient_id, company_id, position, department, is_active,
                start_date, end_date, notes, created_at, created_by
         FROM patient_companies
         WHERE patient_id = $1
         ORDER BY created_at`, patientID)
	return items, err
}

// ── Tutor ─────────────────────────────────────────────────────────────────────

// ListByTutorID returns all patients whose tutor_id matches the given UUID.
func (r *PatientRepository) ListByTutorID(ctx context.Context, tutorID uuid.UUID) ([]*patient.Patient, error) {
	var items []*patient.Patient
	err := r.q.SelectContext(ctx, &items,
		`SELECT p.id, p.rut, p.first_name, p.last_name, p.birth_date, p.status,
                p.patient_type, p.tutor_id, p.created_at,
                `+hasLoginExpr+`
         FROM patients p
         WHERE p.tutor_id = $1 AND p.deleted_at IS NULL
         ORDER BY p.first_name`, tutorID)
	return items, err
}

// ── Login linkage ─────────────────────────────────────────────────────────────

// GetLinkedUserID returns the ID of the active user account linked to the
// given patient, or nil if no such account exists.
func (r *PatientRepository) GetLinkedUserID(ctx context.Context, patientID uuid.UUID) (*uuid.UUID, error) {
	var userID uuid.UUID
	err := r.q.GetContext(ctx, &userID,
		`SELECT id FROM users WHERE patient_id = $1 AND deleted_at IS NULL LIMIT 1`, patientID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	return &userID, nil
}

// ── Clinical record ───────────────────────────────────────────────────────────

func (r *PatientRepository) CreateClinicalRecord(ctx context.Context, cr *patient.ClinicalRecord) error {
	_, err := r.q.ExecContext(ctx,
		`INSERT INTO clinical_records (id, patient_id, consent_signed, created_at, created_by)
         VALUES ($1,$2,$3,$4,$5)`,
		cr.ID, cr.PatientID, cr.ConsentSigned, cr.CreatedAt, cr.CreatedBy)
	return err
}

func (r *PatientRepository) GetClinicalRecord(ctx context.Context, patientID uuid.UUID) (*patient.ClinicalRecord, error) {
	cr := &patient.ClinicalRecord{}
	err := r.q.GetContext(ctx, cr,
		`SELECT cr.id, cr.patient_id, cr.main_diagnosis, cr.allergies, cr.current_medications, cr.relevant_history,
                cr.family_history, cr.physical_restrictions, cr.alerts, cr.occupation,
                cr.consent_signed, cr.consent_date, cr.consent_version,
                cr.created_at, cr.updated_at, cr.created_by, cr.updated_by,
                (uc.first_name || ' ' || uc.last_name) AS created_by_name,
                (uu.first_name || ' ' || uu.last_name) AS updated_by_name
         FROM clinical_records cr
         LEFT JOIN users uc ON uc.id = cr.created_by
         LEFT JOIN users uu ON uu.id = cr.updated_by
         WHERE cr.patient_id = $1`, patientID)
	return cr, err
}

func (r *PatientRepository) UpdateClinicalRecord(ctx context.Context, cr *patient.ClinicalRecord) error {
	_, err := r.q.ExecContext(ctx,
		`UPDATE clinical_records SET
         main_diagnosis=$1, allergies=$2, current_medications=$3, relevant_history=$4,
         family_history=$5, physical_restrictions=$6, alerts=$7, occupation=$8,
         consent_signed=$9, consent_date=$10, consent_version=$11,
         updated_at=$12, updated_by=$13
         WHERE patient_id=$14`,
		cr.MainDiagnosis, cr.Allergies, cr.CurrentMedications, cr.RelevantHistory,
		cr.FamilyHistory, cr.PhysicalRestrictions, cr.Alerts, cr.Occupation,
		cr.ConsentSigned, cr.ConsentDate, cr.ConsentVersion,
		cr.UpdatedAt, cr.UpdatedBy, cr.PatientID)
	return err
}

// AnotherPatientHasEmail returns true when the given email (case-insensitive) is
// stored as the clinical contact email on any active patient OTHER than excludingPatientID.
// This prevents reusing another patient's contact email as a login credential.
func (r *PatientRepository) AnotherPatientHasEmail(ctx context.Context, email string, excludingPatientID uuid.UUID) (bool, error) {
	var count int
	err := r.q.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM patients
         WHERE LOWER(TRIM(email)) = LOWER(TRIM($1))
           AND id <> $2
           AND deleted_at IS NULL
         LIMIT 1`,
		email, excludingPatientID)
	return count > 0, err
}
