package patient

import (
	"time"

	"github.com/google/uuid"
)

// ── Input DTOs ─────────────────────────────────────────────────────────────

type CreatePatientRequest struct {
	RUT       *string `json:"rut"`
	FirstName string  `json:"first_name"  validate:"required,min=2"`
	LastName  string  `json:"last_name"   validate:"required,min=2"`
	BirthDate *string `json:"birth_date"`
	Gender    *string `json:"gender"      validate:"omitempty,oneof=masculino femenino otro prefiero_no_decir"`
	// Email is the clinical/contact email for records and notifications ONLY.
	// It is NOT used as an authentication credential (see Login field for that).
	Email                 *string `json:"email"       validate:"omitempty,email"`
	Phone                 *string `json:"phone"`
	Address               *string `json:"address"`
	City                  *string `json:"city"`
	Region                *string `json:"region"`
	EmergencyContactName  *string `json:"emergency_contact_name"`
	EmergencyContactPhone *string `json:"emergency_contact_phone"`
	GeneralNotes          *string `json:"general_notes"`
	PatientType           string  `json:"patient_type" validate:"required,oneof=particular company both"`
	// TutorID links this patient to an existing patient who acts as their guardian.
	// Recommended when the patient is under 18 years old.
	// The referenced patient must be an adult (>= 18). Cannot be self-referential.
	TutorID   *uuid.UUID           `json:"tutor_id"`
	Companies []CompanyAssociation `json:"companies"`
	// Login optionally creates a portal user account at the same time.
	// Omit to create the patient without login (can be enabled later via POST /patients/{id}/login).
	Login *EnableLoginRequest `json:"login"`
}

type CompanyAssociation struct {
	CompanyID  uuid.UUID `json:"company_id"  validate:"required"`
	Position   *string   `json:"position"`
	Department *string   `json:"department"`
	StartDate  *string   `json:"start_date"`
	Notes      *string   `json:"notes"`
}

type UpdatePatientRequest struct {
	RUT                   *string `json:"rut"`
	FirstName             *string `json:"first_name"  validate:"omitempty,min=2"`
	LastName              *string `json:"last_name"   validate:"omitempty,min=2"`
	BirthDate             *string `json:"birth_date"`
	Gender                *string `json:"gender"      validate:"omitempty,oneof=masculino femenino otro prefiero_no_decir"`
	Email                 *string `json:"email"       validate:"omitempty,email"`
	Phone                 *string `json:"phone"`
	Address               *string `json:"address"`
	City                  *string `json:"city"`
	Region                *string `json:"region"`
	EmergencyContactName  *string `json:"emergency_contact_name"`
	EmergencyContactPhone *string `json:"emergency_contact_phone"`
	GeneralNotes          *string `json:"general_notes"`
	PatientType           *string `json:"patient_type" validate:"omitempty,oneof=particular company both"`
	Status                *string `json:"status"       validate:"omitempty,oneof=active inactive discharged"`
	// TutorID replaces the current tutor. Send uuid.Nil ("00000000-...") to clear.
	// Use ClearTutor=true as an explicit alternative.
	TutorID *uuid.UUID `json:"tutor_id"`
	// ClearTutor removes any existing tutor association when true.
	ClearTutor bool `json:"clear_tutor"`
	// Companies replaces ALL current associations:
	//   - omit  → don't touch companies
	//   - []    → remove all companies
	//   - [...] → replace with these companies
	Companies *[]CompanyAssociation `json:"companies"`
}

// EnableLoginRequest creates or re-enables the portal user account for a patient.
type EnableLoginRequest struct {
	// LoginEmail is the AUTHENTICATION email — must be unique across all users.
	// It does NOT have to match the patient's clinical contact email.
	LoginEmail    string `json:"login_email"    validate:"required,email"`
	LoginPassword string `json:"login_password" validate:"required,min=8"`
}

type UpdateClinicalRecordRequest struct {
	MainDiagnosis        *string `json:"main_diagnosis"`
	Allergies            *string `json:"allergies"`
	CurrentMedications   *string `json:"current_medications"`
	RelevantHistory      *string `json:"relevant_history"`
	FamilyHistory        *string `json:"family_history"`
	PhysicalRestrictions *string `json:"physical_restrictions"`
	Alerts               *string `json:"alerts"`
	Occupation           *string `json:"occupation"`
	ConsentSigned        *bool   `json:"consent_signed"`
	ConsentDate          *string `json:"consent_date"`
	ConsentVersion       *string `json:"consent_version"`
}

type PatientFilters struct {
	Search          string
	Status          string
	PatientType     string
	CompanyID       *uuid.UUID
	FollowUpPending bool
}

// ── Output DTOs ────────────────────────────────────────────────────────────

type PatientListItem struct {
	ID          uuid.UUID  `json:"id"`
	RUT         *string    `json:"rut,omitempty"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	Email       *string    `json:"email,omitempty"`
	Phone       *string    `json:"phone,omitempty"`
	PatientType string     `json:"patient_type"`
	Status      string     `json:"status"`
	TutorID     *uuid.UUID `json:"tutor_id,omitempty"`
	HasLogin    bool       `json:"has_login"`
	CreatedAt   time.Time  `json:"created_at"`
}

// PatientDetailResponse is the full representation of a patient including all
// related data: company associations, tutor info, wards (patients this person
// tutors), and login status.
type PatientDetailResponse struct {
	ID                    uuid.UUID                  `json:"id"`
	RUT                   *string                    `json:"rut,omitempty"`
	FirstName             string                     `json:"first_name"`
	LastName              string                     `json:"last_name"`
	BirthDate             *time.Time                 `json:"birth_date,omitempty"`
	Gender                *string                    `json:"gender,omitempty"`
	Email                 *string                    `json:"email,omitempty"`
	Phone                 *string                    `json:"phone,omitempty"`
	Address               *string                    `json:"address,omitempty"`
	City                  *string                    `json:"city,omitempty"`
	Region                *string                    `json:"region,omitempty"`
	EmergencyContactName  *string                    `json:"emergency_contact_name,omitempty"`
	EmergencyContactPhone *string                    `json:"emergency_contact_phone,omitempty"`
	GeneralNotes          *string                    `json:"general_notes,omitempty"`
	PatientType           string                     `json:"patient_type"`
	Status                string                     `json:"status"`
	HasLogin              bool                       `json:"has_login"`
	CreatedAt             time.Time                  `json:"created_at"`
	UpdatedAt             *time.Time                 `json:"updated_at,omitempty"`
	Companies             []CompanyAssociationDetail `json:"companies"`
	Tutor                 *TutorInfo                 `json:"tutor,omitempty"`
	Wards                 []TutorInfo                `json:"wards"`
}

type CompanyAssociationDetail struct {
	ID         uuid.UUID  `json:"id"`
	CompanyID  uuid.UUID  `json:"company_id"`
	Position   *string    `json:"position,omitempty"`
	Department *string    `json:"department,omitempty"`
	IsActive   bool       `json:"is_active"`
	StartDate  *time.Time `json:"start_date,omitempty"`
	EndDate    *time.Time `json:"end_date,omitempty"`
	Notes      *string    `json:"notes,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// TutorInfo is a compact patient summary used inside PatientDetailResponse.
type TutorInfo struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	RUT       *string   `json:"rut,omitempty"`
}

// PatientLoginInfo is returned by GET /patients/{id}/login.
// It gives the edit form the current portal auth state of the patient.
type PatientLoginInfo struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	IsActive  bool      `json:"is_active"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
}
