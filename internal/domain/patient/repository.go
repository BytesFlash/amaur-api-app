package patient

import (
	"context"

	"github.com/google/uuid"
)

type Filter struct {
	Search          string
	Status          string
	PatientType     string
	CompanyID       *uuid.UUID
	WorkerID        *uuid.UUID
	FollowUpPending bool
}

// Repository defines all persistence operations for the patient aggregate.
//
// Company associations use a full-replace strategy (ReplaceCompanies):
// pass the desired final set on every write — the implementation deletes
// existing rows and inserts the new ones within the same transaction.
type Repository interface {
	Create(ctx context.Context, p *Patient) error
	FindByID(ctx context.Context, id uuid.UUID) (*Patient, error)
	FindByRUT(ctx context.Context, rut string) (*Patient, error)
	Update(ctx context.Context, p *Patient) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f Filter, limit, offset int) ([]*Patient, int64, error)

	// Company associations — full replace on every update.
	ReplaceCompanies(ctx context.Context, patientID uuid.UUID, companies []*PatientCompany) error
	ListPatientCompanies(ctx context.Context, patientID uuid.UUID) ([]*PatientCompany, error)

	// Tutor (guardian) relationship.
	ListByTutorID(ctx context.Context, tutorID uuid.UUID) ([]*Patient, error)

	// Login linkage: returns the user.ID linked to this patient, or nil if none.
	GetLinkedUserID(ctx context.Context, patientID uuid.UUID) (*uuid.UUID, error)

	// AnotherPatientHasEmail returns true when the given email is already stored
	// as a clinical contact email on a different patient (excludingID) who is not soft-deleted.
	// Used to prevent using another patient's contact email as a login credential.
	AnotherPatientHasEmail(ctx context.Context, email string, excludingPatientID uuid.UUID) (bool, error)

	// Clinical record.
	CreateClinicalRecord(ctx context.Context, cr *ClinicalRecord) error
	GetClinicalRecord(ctx context.Context, patientID uuid.UUID) (*ClinicalRecord, error)
	UpdateClinicalRecord(ctx context.Context, cr *ClinicalRecord) error

	// InTx executes fn inside a database transaction.
	// If fn returns an error the transaction is rolled back; otherwise committed.
	// Calling InTx on a repository that is already inside a transaction is a
	// no-op (the existing transaction is reused).
	InTx(ctx context.Context, fn func(tx Repository) error) error
}
