package patient

import (
	"time"

	"github.com/google/uuid"
)

type PatientType string
type PatientStatus string

const (
	PatientTypeParticular PatientType = "particular"
	PatientTypeCompany    PatientType = "company"
	PatientTypeBoth       PatientType = "both"

	PatientStatusActive     PatientStatus = "active"
	PatientStatusInactive   PatientStatus = "inactive"
	PatientStatusDischarged PatientStatus = "discharged"
)

// Patient is the clinical identity of a person receiving care.
//
// IMPORTANT — two email concepts exist in this system:
//   - Patient.Email    → contact/clinical email stored in the patients table.
//     Used for scheduling notifications, not for authentication.
//   - users.email      → authentication email, unique across the system.
//     A patient MAY have a linked users row (users.patient_id = patient.id)
//     that gives them portal access.
//
// A patient can exist without any user account (no login required).
// A patient may designate another patient as their TutorID (guardian).
// Minors (< 18 years old) may be associated with a tutor.
type Patient struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	RUT       *string    `db:"rut" json:"rut,omitempty"`
	FirstName string     `db:"first_name" json:"first_name"`
	LastName  string     `db:"last_name" json:"last_name"`
	BirthDate *time.Time `db:"birth_date" json:"birth_date,omitempty"`
	Gender    *string    `db:"gender" json:"gender,omitempty"`
	// Email is the clinical/contact email only — NOT an authentication credential.
	Email                 *string       `db:"email" json:"email,omitempty"`
	Phone                 *string       `db:"phone" json:"phone,omitempty"`
	Address               *string       `db:"address" json:"address,omitempty"`
	City                  *string       `db:"city" json:"city,omitempty"`
	Region                *string       `db:"region" json:"region,omitempty"`
	EmergencyContactName  *string       `db:"emergency_contact_name" json:"emergency_contact_name,omitempty"`
	EmergencyContactPhone *string       `db:"emergency_contact_phone" json:"emergency_contact_phone,omitempty"`
	GeneralNotes          *string       `db:"general_notes" json:"general_notes,omitempty"`
	PatientType           PatientType   `db:"patient_type" json:"patient_type"`
	Status                PatientStatus `db:"status" json:"status"`
	// TutorID references another Patient who acts as this patient's guardian.
	// Self-reference is forbidden at DB level (CHECK constraint).
	TutorID *uuid.UUID `db:"tutor_id" json:"tutor_id,omitempty"`
	// HasLogin is computed in queries; it is NOT a column in the patients table.
	HasLogin  bool       `db:"has_login" json:"has_login"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at,omitempty"`
	CreatedBy *uuid.UUID `db:"created_by" json:"created_by,omitempty"`
	UpdatedBy *uuid.UUID `db:"updated_by" json:"updated_by,omitempty"`
	DeletedAt *time.Time `db:"deleted_at" json:"-"`
}

func (p *Patient) FullName() string {
	return p.FirstName + " " + p.LastName
}

// IsMinor returns true when the patient's age is below 18 years.
func (p *Patient) IsMinor() bool {
	if p.BirthDate == nil {
		return false
	}
	adultCutoff := time.Now().AddDate(-18, 0, 0)
	return p.BirthDate.After(adultCutoff)
}

type PatientCompany struct {
	ID         uuid.UUID  `db:"id" json:"id"`
	PatientID  uuid.UUID  `db:"patient_id" json:"patient_id"`
	CompanyID  uuid.UUID  `db:"company_id" json:"company_id"`
	Position   *string    `db:"position" json:"position,omitempty"`
	Department *string    `db:"department" json:"department,omitempty"`
	IsActive   bool       `db:"is_active" json:"is_active"`
	StartDate  *time.Time `db:"start_date" json:"start_date,omitempty"`
	EndDate    *time.Time `db:"end_date" json:"end_date,omitempty"`
	Notes      *string    `db:"notes" json:"notes,omitempty"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	CreatedBy  *uuid.UUID `db:"created_by" json:"created_by,omitempty"`
}

type ClinicalRecord struct {
	ID                   uuid.UUID  `db:"id" json:"id"`
	PatientID            uuid.UUID  `db:"patient_id" json:"patient_id"`
	MainDiagnosis        *string    `db:"main_diagnosis" json:"main_diagnosis,omitempty"`
	Allergies            *string    `db:"allergies" json:"allergies,omitempty"`
	CurrentMedications   *string    `db:"current_medications" json:"current_medications,omitempty"`
	RelevantHistory      *string    `db:"relevant_history" json:"relevant_history,omitempty"`
	FamilyHistory        *string    `db:"family_history" json:"family_history,omitempty"`
	PhysicalRestrictions *string    `db:"physical_restrictions" json:"physical_restrictions,omitempty"`
	Alerts               *string    `db:"alerts" json:"alerts,omitempty"`
	Occupation           *string    `db:"occupation" json:"occupation,omitempty"`
	ConsentSigned        bool       `db:"consent_signed" json:"consent_signed"`
	ConsentDate          *time.Time `db:"consent_date" json:"consent_date,omitempty"`
	ConsentVersion       *string    `db:"consent_version" json:"consent_version,omitempty"`
	CreatedAt            time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt            *time.Time `db:"updated_at" json:"updated_at,omitempty"`
	CreatedBy            *uuid.UUID `db:"created_by" json:"created_by,omitempty"`
	UpdatedBy            *uuid.UUID `db:"updated_by" json:"updated_by,omitempty"`
	// CreatedByName and UpdatedByName are populated only by GetClinicalRecord (JOIN on users).
	CreatedByName *string `db:"created_by_name" json:"created_by_name,omitempty"`
	UpdatedByName *string `db:"updated_by_name" json:"updated_by_name,omitempty"`
}
