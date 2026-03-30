package patient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domaincompany "amaur/api/internal/domain/company"
	domainpatient "amaur/api/internal/domain/patient"
	domainuser "amaur/api/internal/domain/user"
	"amaur/api/pkg/password"

	"github.com/google/uuid"
)

var (
	ErrPatientNotFound    = errors.New("patient not found")
	ErrDuplicateRUT       = errors.New("a patient with this RUT already exists")
	ErrTutorNotFound      = errors.New("tutor patient not found")
	ErrTutorMustBeAdult   = errors.New("a tutor must be 18 years old or older")
	ErrSelfTutor          = errors.New("a patient cannot be their own tutor")
	ErrLoginExists        = errors.New("this patient already has an active login")
	ErrNoLogin            = errors.New("this patient does not have a login")
	ErrEmailTaken         = errors.New("this email is already used by another user account")
	ErrLoginEmailRequired = errors.New("login_email is required: the patient has no clinical email to use as fallback")
	// ErrEmailUsedByAnotherPatient is returned when the requested login email is
	// already registered as the clinical/contact email of a different patient.
	// Each patient must have a distinct email identity.
	ErrEmailUsedByAnotherPatient = errors.New("this email is registered as the clinical email of another patient")
	// ErrMinorRequiresTutor is returned when a patient with a confirmed birth_date
	// that places them under 18 is created or updated without a tutor reference.
	// The tutor must be a registered adult patient.
	ErrMinorRequiresTutor = errors.New("patients under 18 must be linked to an adult tutor")
	ErrNotFound           = errors.New("record not found")
	ErrDuplicateCompanyID = errors.New("duplicate company_id in payload")
)

// ErrInvalidCompanies is returned when one or more company IDs in the
// payload do not correspond to active companies.
type ErrInvalidCompanies struct {
	MissingIDs []uuid.UUID
}

func (e *ErrInvalidCompanies) Error() string {
	ids := make([]string, len(e.MissingIDs))
	for i, id := range e.MissingIDs {
		ids[i] = id.String()
	}
	return "companies not found: " + strings.Join(ids, ", ")
}

// Service orchestrates patient use-cases.
// It owns patient data AND the lifecycle of the optional user-login linked to a patient.
type Service struct {
	repo        domainpatient.Repository
	userRepo    domainuser.Repository
	companyRepo domaincompany.Repository
}

func NewService(repo domainpatient.Repository, userRepo domainuser.Repository, companyRepo domaincompany.Repository) *Service {
	return &Service{repo: repo, userRepo: userRepo, companyRepo: companyRepo}
}

// ── Create ───────────────────────────────────────────────────────────────────

// Create registers a new patient inside a single transaction:
//
//	patient row + empty clinical record + company associations.
//
// If a Login block is provided, a user account is created AFTER the
// transaction (see enableLogin for the rationale).
func (s *Service) Create(ctx context.Context, req CreatePatientRequest, createdBy uuid.UUID) (*PatientDetailResponse, error) {
	// 1. RUT uniqueness
	if req.RUT != nil && strings.TrimSpace(*req.RUT) != "" {
		existing, err := s.repo.FindByRUT(ctx, *req.RUT)
		if err == nil && existing != nil {
			return nil, ErrDuplicateRUT
		}
	}

	// 2. Tutor validation
	if err := s.validateTutor(ctx, req.TutorID, uuid.Nil); err != nil {
		return nil, err
	}

	// 3. Company payload: deduplication + existence
	if err := s.validateCompanyPayload(ctx, req.Companies); err != nil {
		return nil, err
	}

	now := time.Now()
	p := &domainpatient.Patient{
		ID:          uuid.New(),
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		PatientType: domainpatient.PatientType(req.PatientType),
		Status:      domainpatient.PatientStatusActive,
		TutorID:     req.TutorID,
		CreatedAt:   now,
		CreatedBy:   &createdBy,
	}
	if req.RUT != nil && strings.TrimSpace(*req.RUT) != "" {
		p.RUT = req.RUT
	}
	if req.BirthDate != nil && *req.BirthDate != "" {
		if t, err := time.Parse("2006-01-02", *req.BirthDate); err == nil {
			p.BirthDate = &t
		}
	}
	p.Gender = req.Gender
	p.Email = req.Email
	p.Phone = req.Phone
	p.Address = req.Address
	p.City = req.City
	p.Region = req.Region
	p.EmergencyContactName = req.EmergencyContactName
	p.EmergencyContactPhone = req.EmergencyContactPhone
	p.GeneralNotes = req.GeneralNotes

	companies := buildPatientCompanies(p.ID, req.Companies, &createdBy, now)

	// 4a. Minor tutor enforcement: when a known birth_date confirms the patient
	// is under 18, a tutor (adult registered patient) is mandatory.
	if p.IsMinor() && p.TutorID == nil {
		return nil, ErrMinorRequiresTutor
	}

	// 4. Atomic: patient + clinical record + companies
	err := s.repo.InTx(ctx, func(tx domainpatient.Repository) error {
		if err := tx.Create(ctx, p); err != nil {
			return err
		}
		cr := &domainpatient.ClinicalRecord{
			ID:        uuid.New(),
			PatientID: p.ID,
			CreatedAt: now,
			CreatedBy: &createdBy,
		}
		if err := tx.CreateClinicalRecord(ctx, cr); err != nil {
			return err
		}
		return tx.ReplaceCompanies(ctx, p.ID, companies)
	})
	if err != nil {
		return nil, err
	}

	// 5. Optional login (separate step; patient is already persisted).
	if req.Login != nil {
		login := *req.Login
		// Autocomplete: use the patient's clinical email when login_email is omitted.
		if strings.TrimSpace(login.LoginEmail) == "" && p.Email != nil && strings.TrimSpace(*p.Email) != "" {
			login.LoginEmail = *p.Email
		}
		if err := s.enableLogin(ctx, p.ID, p.FirstName, p.LastName, login, createdBy); err != nil {
			// Compensate: soft-delete the patient so the system stays consistent.
			_ = s.repo.SoftDelete(ctx, p.ID)
			return nil, fmt.Errorf("patient creation rolled back because login setup failed: %w", err)
		}
	}

	// 6. Return full detail (includes companies list so the caller sees what was saved).
	return s.GetDetail(ctx, p.ID)
}

// ── Read ────────────────────────────────────────────────────────────────────

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*domainpatient.Patient, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrPatientNotFound
	}
	return p, nil
}

// GetDetail returns the full patient detail including companies, tutor and wards.
func (s *Service) GetDetail(ctx context.Context, id uuid.UUID) (*PatientDetailResponse, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrPatientNotFound
	}

	companies, _ := s.repo.ListPatientCompanies(ctx, id)
	wards, _ := s.repo.ListByTutorID(ctx, id)

	resp := &PatientDetailResponse{
		ID:                    p.ID,
		RUT:                   p.RUT,
		FirstName:             p.FirstName,
		LastName:              p.LastName,
		BirthDate:             p.BirthDate,
		Gender:                p.Gender,
		Email:                 p.Email,
		Phone:                 p.Phone,
		Address:               p.Address,
		City:                  p.City,
		Region:                p.Region,
		EmergencyContactName:  p.EmergencyContactName,
		EmergencyContactPhone: p.EmergencyContactPhone,
		GeneralNotes:          p.GeneralNotes,
		PatientType:           string(p.PatientType),
		Status:                string(p.Status),
		HasLogin:              p.HasLogin,
		CreatedAt:             p.CreatedAt,
		UpdatedAt:             p.UpdatedAt,
		Companies:             toCompanyDetails(companies),
		Wards:                 toTutorInfoList(wards),
	}

	if p.TutorID != nil {
		tutor, err := s.repo.FindByID(ctx, *p.TutorID)
		if err == nil {
			resp.Tutor = &TutorInfo{
				ID:        tutor.ID,
				FirstName: tutor.FirstName,
				LastName:  tutor.LastName,
				RUT:       tutor.RUT,
			}
		}
	}

	return resp, nil
}

func (s *Service) List(ctx context.Context, f PatientFilters, limit, offset int) ([]*PatientListItem, int64, error) {
	df := domainpatient.Filter{
		Search:          f.Search,
		Status:          f.Status,
		PatientType:     f.PatientType,
		CompanyID:       f.CompanyID,
		FollowUpPending: f.FollowUpPending,
	}
	patients, total, err := s.repo.List(ctx, df, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*PatientListItem, 0, len(patients))
	for _, p := range patients {
		items = append(items, toListItem(p))
	}
	return items, total, nil
}

func (s *Service) GetCompanies(ctx context.Context, patientID uuid.UUID) ([]CompanyAssociationDetail, error) {
	if _, err := s.repo.FindByID(ctx, patientID); err != nil {
		return nil, ErrPatientNotFound
	}
	companies, err := s.repo.ListPatientCompanies(ctx, patientID)
	if err != nil {
		return nil, err
	}
	return toCompanyDetails(companies), nil
}

// GetWards returns the list of patients (wards) for whom the given patient
// acts as a tutor/guardian. Returns ErrPatientNotFound when the tutor patient
// itself does not exist. Returns an empty slice (not an error) when the tutor
// has no wards registered yet.
func (s *Service) GetWards(ctx context.Context, tutorID uuid.UUID) ([]TutorInfo, error) {
	if _, err := s.repo.FindByID(ctx, tutorID); err != nil {
		return nil, ErrPatientNotFound
	}
	wards, err := s.repo.ListByTutorID(ctx, tutorID)
	if err != nil {
		return nil, err
	}
	return toTutorInfoList(wards), nil
}

// ── Update ───────────────────────────────────────────────────────────────────

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdatePatientRequest, updatedBy uuid.UUID) (*PatientDetailResponse, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrPatientNotFound
	}

	// Company payload: deduplication + existence (validate before any DB writes)
	if req.Companies != nil {
		if err := s.validateCompanyPayload(ctx, *req.Companies); err != nil {
			return nil, err
		}
	}

	// Determine desired tutor
	desiredTutor := p.TutorID
	if req.ClearTutor {
		desiredTutor = nil
	} else if req.TutorID != nil {
		if *req.TutorID == uuid.Nil {
			desiredTutor = nil
		} else {
			desiredTutor = req.TutorID
		}
	}
	if err := s.validateTutor(ctx, desiredTutor, id); err != nil {
		return nil, err
	}

	now := time.Now()
	p.UpdatedAt = &now
	p.UpdatedBy = &updatedBy
	p.TutorID = desiredTutor

	if req.FirstName != nil {
		p.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		p.LastName = *req.LastName
	}
	if req.RUT != nil {
		if strings.TrimSpace(*req.RUT) == "" {
			p.RUT = nil
		} else {
			p.RUT = req.RUT
		}
	}
	if req.BirthDate != nil {
		if *req.BirthDate == "" {
			p.BirthDate = nil
		} else if t, err := time.Parse("2006-01-02", *req.BirthDate); err == nil {
			p.BirthDate = &t
		}
	}
	if req.Gender != nil {
		p.Gender = req.Gender
	}
	if req.Email != nil {
		p.Email = req.Email
	}
	if req.Phone != nil {
		p.Phone = req.Phone
	}
	if req.Address != nil {
		p.Address = req.Address
	}
	if req.City != nil {
		p.City = req.City
	}
	if req.Region != nil {
		p.Region = req.Region
	}
	if req.EmergencyContactName != nil {
		p.EmergencyContactName = req.EmergencyContactName
	}
	if req.EmergencyContactPhone != nil {
		p.EmergencyContactPhone = req.EmergencyContactPhone
	}
	if req.GeneralNotes != nil {
		p.GeneralNotes = req.GeneralNotes
	}
	if req.PatientType != nil {
		p.PatientType = domainpatient.PatientType(*req.PatientType)
	}
	if req.Status != nil {
		p.Status = domainpatient.PatientStatus(*req.Status)
	}

	// Minor tutor enforcement: applies to the final state after all field updates.
	if p.IsMinor() && p.TutorID == nil {
		return nil, ErrMinorRequiresTutor
	}

	// Atomic: update patient + replace companies (both or neither)
	err = s.repo.InTx(ctx, func(tx domainpatient.Repository) error {
		if err := tx.Update(ctx, p); err != nil {
			return err
		}
		if req.Companies != nil {
			companies := buildPatientCompanies(p.ID, *req.Companies, &updatedBy, now)
			return tx.ReplaceCompanies(ctx, p.ID, companies)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Return full detail (includes updated companies list).
	return s.GetDetail(ctx, p.ID)
}

// ── Delete ───────────────────────────────────────────────────────────────────

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return ErrPatientNotFound
	}
	return s.repo.SoftDelete(ctx, id)
}

// ── Login management ─────────────────────────────────────────────────────────

// EnableLogin creates (or reactivates) a portal user account for an existing patient.
//
// Behaviour matrix:
//   - Patient has active login            → ErrLoginExists
//   - Patient had login (now disabled)    → reactivates the existing user record
//   - login_email omitted, patient has clinical email → clinical email is used automatically
//   - login_email omitted, no clinical email          → ErrLoginEmailRequired
//   - login_email belongs to another user             → ErrEmailTaken
func (s *Service) EnableLogin(ctx context.Context, patientID uuid.UUID, req EnableLoginRequest, createdBy uuid.UUID) error {
	p, err := s.repo.FindByID(ctx, patientID)
	if err != nil {
		return ErrPatientNotFound
	}

	// Autocomplete: fall back to the patient's clinical email when login_email is not supplied.
	if strings.TrimSpace(req.LoginEmail) == "" {
		if p.Email == nil || strings.TrimSpace(*p.Email) == "" {
			return ErrLoginEmailRequired
		}
		req.LoginEmail = *p.Email
	}

	return s.enableLogin(ctx, p.ID, p.FirstName, p.LastName, req, createdBy)
}

// DisableLogin soft-deletes the user account linked to a patient and revokes
// all active refresh tokens so sessions are terminated immediately.
func (s *Service) DisableLogin(ctx context.Context, patientID uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, patientID); err != nil {
		return ErrPatientNotFound
	}
	linkedUID, err := s.repo.GetLinkedUserID(ctx, patientID)
	if err != nil {
		return err
	}
	if linkedUID == nil {
		return ErrNoLogin
	}
	if err := s.userRepo.RevokeAllRefreshTokens(ctx, *linkedUID); err != nil {
		return err
	}
	return s.userRepo.SoftDelete(ctx, *linkedUID)
}

// GetLoginInfo returns the login/user account summary for a patient.
// Used by the edit form so operators can see the current auth email.
// Returns ErrPatientNotFound when the patient does not exist.
// Returns ErrNoLogin when the patient has no active portal account.
func (s *Service) GetLoginInfo(ctx context.Context, patientID uuid.UUID) (*PatientLoginInfo, error) {
	if _, err := s.repo.FindByID(ctx, patientID); err != nil {
		return nil, ErrPatientNotFound
	}
	u, err := s.userRepo.FindByPatientID(ctx, patientID)
	if err != nil {
		// No active user linked.
		return nil, ErrNoLogin
	}
	roles, _ := s.userRepo.GetRoleNames(ctx, u.ID)
	return &PatientLoginInfo{
		UserID:    u.ID,
		Email:     u.Email,
		IsActive:  u.IsActive,
		Roles:     roles,
		CreatedAt: u.CreatedAt,
	}, nil
}

// ── Clinical record ──────────────────────────────────────────────────────────

func (s *Service) GetClinicalRecord(ctx context.Context, patientID uuid.UUID) (*domainpatient.ClinicalRecord, error) {
	return s.repo.GetClinicalRecord(ctx, patientID)
}

func (s *Service) UpdateClinicalRecord(ctx context.Context, patientID uuid.UUID, req UpdateClinicalRecordRequest, updatedBy uuid.UUID) (*domainpatient.ClinicalRecord, error) {
	cr, err := s.repo.GetClinicalRecord(ctx, patientID)
	if err != nil {
		return nil, ErrNotFound
	}

	now := time.Now()
	cr.UpdatedAt = &now
	cr.UpdatedBy = &updatedBy

	cr.MainDiagnosis = req.MainDiagnosis
	cr.Allergies = req.Allergies
	cr.CurrentMedications = req.CurrentMedications
	cr.RelevantHistory = req.RelevantHistory
	cr.FamilyHistory = req.FamilyHistory
	cr.PhysicalRestrictions = req.PhysicalRestrictions
	cr.Alerts = req.Alerts
	cr.Occupation = req.Occupation

	if req.ConsentSigned != nil {
		cr.ConsentSigned = *req.ConsentSigned
		if *req.ConsentSigned && req.ConsentDate != nil {
			if t, err := time.Parse("2006-01-02", *req.ConsentDate); err == nil {
				cr.ConsentDate = &t
			}
		}
	}
	if req.ConsentVersion != nil {
		cr.ConsentVersion = req.ConsentVersion
	}

	if err := s.repo.UpdateClinicalRecord(ctx, cr); err != nil {
		return nil, err
	}
	return cr, nil
}

// ── Private helpers ──────────────────────────────────────────────────────────

// validateCompanyPayload checks the list of company associations for two problems
// before any DB writes happen:
//  1. Duplicate company_id within the same request (would cause a unique-constraint failure).
//  2. company_id values that do not exist as active companies in the database.
//
// Returns ErrDuplicateCompanyID or *ErrInvalidCompanies on failure.
func (s *Service) validateCompanyPayload(ctx context.Context, associations []CompanyAssociation) error {
	if len(associations) == 0 {
		return nil
	}

	// Deduplication check
	seen := make(map[uuid.UUID]struct{}, len(associations))
	ids := make([]uuid.UUID, 0, len(associations))
	for _, ca := range associations {
		if _, dup := seen[ca.CompanyID]; dup {
			return ErrDuplicateCompanyID
		}
		seen[ca.CompanyID] = struct{}{}
		ids = append(ids, ca.CompanyID)
	}

	// Existence check
	missing, err := s.companyRepo.ExistsByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return &ErrInvalidCompanies{MissingIDs: missing}
	}
	return nil
}

// validateTutor enforces business rules on tutor assignment.
// selfID is the ID of the patient being edited (uuid.Nil on create).
func (s *Service) validateTutor(ctx context.Context, tutorID *uuid.UUID, selfID uuid.UUID) error {
	if tutorID == nil || *tutorID == uuid.Nil {
		return nil
	}
	if selfID != uuid.Nil && *tutorID == selfID {
		return ErrSelfTutor
	}
	tutor, err := s.repo.FindByID(ctx, *tutorID)
	if err != nil {
		return ErrTutorNotFound
	}
	if tutor.IsMinor() {
		return ErrTutorMustBeAdult
	}
	return nil
}

// enableLogin is the shared login-creation/reactivation logic called by both
// Create and EnableLogin. It handles four distinct cases:
//
//  1. Patient already has an active user account → ErrLoginExists.
//  2. Patient had an account that was soft-deleted (via DisableLogin) → reactivate it.
//     If the requested email differs from the stored one, verify it is not taken.
//  3. No prior account exists but the email is claimed by another active user → ErrEmailTaken.
//  4. Clean slate → create a new user account and assign the "patient" role.
func (s *Service) enableLogin(
	ctx context.Context,
	patientID uuid.UUID,
	firstName, lastName string,
	req EnableLoginRequest,
	createdBy uuid.UUID,
) error {
	email := strings.ToLower(strings.TrimSpace(req.LoginEmail))

	// Case 1 — already has an active login.
	if activeUser, _ := s.userRepo.FindByPatientID(ctx, patientID); activeUser != nil && activeUser.ID != (uuid.UUID{}) {
		return ErrLoginExists
	}

	// Case 2 — had a login before; reactivate instead of inserting a duplicate.
	softUser, err := s.userRepo.FindSoftDeletedByPatientID(ctx, patientID)
	if err != nil {
		return err
	}
	if softUser != nil {
		// If the caller wants a different email, confirm it is not taken by another active user
		// and not used as clinical email of a different patient.
		if !strings.EqualFold(softUser.Email, email) {
			if conflict, _ := s.userRepo.FindByEmail(ctx, email); conflict != nil {
				return ErrEmailTaken
			}
			if taken, _ := s.repo.AnotherPatientHasEmail(ctx, email, patientID); taken {
				return ErrEmailUsedByAnotherPatient
			}
		}
		hash, err := password.Hash(req.LoginPassword)
		if err != nil {
			return err
		}
		softUser.Email = email
		softUser.PasswordHash = hash
		softUser.FirstName = firstName
		softUser.LastName = lastName
		if err := s.userRepo.Reactivate(ctx, softUser); err != nil {
			return err
		}
		// Re-ensure the "patient" role is still assigned (ON CONFLICT DO NOTHING).
		if roleID, err := s.userRepo.GetRoleIDByName(ctx, "patient"); err == nil {
			_ = s.userRepo.AssignRole(ctx, softUser.ID, roleID, createdBy)
		}
		return nil
	}

	// Case 3 — first-time login; block if the email is already used elsewhere
	// or is the clinical contact email of a different patient.
	if existing, _ := s.userRepo.FindByEmail(ctx, email); existing != nil {
		return ErrEmailTaken
	}
	if taken, _ := s.repo.AnotherPatientHasEmail(ctx, email, patientID); taken {
		return ErrEmailUsedByAnotherPatient
	}

	// Case 4 — create a fresh user account.
	hash, err := password.Hash(req.LoginPassword)
	if err != nil {
		return err
	}

	u := &domainuser.User{
		ID:           uuid.New(),
		Email:        email,
		PatientID:    &patientID,
		PasswordHash: hash,
		FirstName:    firstName,
		LastName:     lastName,
		IsActive:     true,
		CreatedAt:    time.Now(),
		CreatedBy:    &createdBy,
	}
	if err := s.userRepo.Create(ctx, u); err != nil {
		return err
	}

	// Assign the 'patient' portal role (seeded in migration 000017).
	if roleID, err := s.userRepo.GetRoleIDByName(ctx, "patient"); err == nil {
		_ = s.userRepo.AssignRole(ctx, u.ID, roleID, createdBy)
	}
	return nil
}

func buildPatientCompanies(
	patientID uuid.UUID,
	input []CompanyAssociation,
	by *uuid.UUID,
	now time.Time,
) []*domainpatient.PatientCompany {
	pcs := make([]*domainpatient.PatientCompany, 0, len(input))
	for _, ca := range input {
		pc := &domainpatient.PatientCompany{
			ID:         uuid.New(),
			PatientID:  patientID,
			CompanyID:  ca.CompanyID,
			Position:   ca.Position,
			Department: ca.Department,
			Notes:      ca.Notes,
			IsActive:   true,
			CreatedAt:  now,
			CreatedBy:  by,
		}
		if ca.StartDate != nil && *ca.StartDate != "" {
			if t, err := time.Parse("2006-01-02", *ca.StartDate); err == nil {
				pc.StartDate = &t
			}
		}
		pcs = append(pcs, pc)
	}
	return pcs
}

func toListItem(p *domainpatient.Patient) *PatientListItem {
	return &PatientListItem{
		ID:          p.ID,
		RUT:         p.RUT,
		FirstName:   p.FirstName,
		LastName:    p.LastName,
		Email:       p.Email,
		Phone:       p.Phone,
		PatientType: string(p.PatientType),
		Status:      string(p.Status),
		TutorID:     p.TutorID,
		HasLogin:    p.HasLogin,
		CreatedAt:   p.CreatedAt,
	}
}

func toCompanyDetails(pcs []*domainpatient.PatientCompany) []CompanyAssociationDetail {
	out := make([]CompanyAssociationDetail, 0, len(pcs))
	for _, pc := range pcs {
		out = append(out, CompanyAssociationDetail{
			ID:         pc.ID,
			CompanyID:  pc.CompanyID,
			Position:   pc.Position,
			Department: pc.Department,
			IsActive:   pc.IsActive,
			StartDate:  pc.StartDate,
			EndDate:    pc.EndDate,
			Notes:      pc.Notes,
			CreatedAt:  pc.CreatedAt,
		})
	}
	return out
}

func toTutorInfoList(patients []*domainpatient.Patient) []TutorInfo {
	out := make([]TutorInfo, 0, len(patients))
	for _, p := range patients {
		out = append(out, TutorInfo{
			ID:        p.ID,
			FirstName: p.FirstName,
			LastName:  p.LastName,
			RUT:       p.RUT,
		})
	}
	return out
}
