package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appatient "amaur/api/internal/application/patient"
	"amaur/api/internal/delivery/http/middleware"
	"amaur/api/internal/delivery/http/response"
	"amaur/api/pkg/pagination"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type PatientHandler struct {
	svc *appatient.Service
}

func NewPatientHandler(svc *appatient.Service) *PatientHandler {
	return &PatientHandler{svc: svc}
}

// GET /api/v1/patients
func (h *PatientHandler) List(w http.ResponseWriter, r *http.Request) {
	p := pagination.FromRequest(r)
	q := r.URL.Query()

	filters := appatient.PatientFilters{
		Search:          q.Get("search"),
		Status:          q.Get("status"),
		PatientType:     q.Get("patient_type"),
		FollowUpPending: q.Get("follow_up_pending") == "true",
	}
	if cid := q.Get("company_id"); cid != "" {
		if id, err := uuid.Parse(cid); err == nil {
			filters.CompanyID = &id
		}
	}

	items, total, err := h.svc.List(r.Context(), filters, p.Limit, p.Offset)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Paginated(w, items, pagination.NewMeta(p, total))
}

// POST /api/v1/patients
func (h *PatientHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req appatient.CreatePatientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}

	claims := middleware.ClaimsFromContext(r.Context())
	p, err := h.svc.Create(r.Context(), req, claims.UserID)
	if err != nil {
		var invComp *appatient.ErrInvalidCompanies
		switch {
		case errors.Is(err, appatient.ErrDuplicateRUT):
			response.Conflict(w, "DUPLICATE_RUT", "A patient with this RUT already exists")
		case errors.Is(err, appatient.ErrTutorNotFound):
			response.BadRequest(w, "TUTOR_NOT_FOUND", "The specified tutor patient does not exist")
		case errors.Is(err, appatient.ErrTutorMustBeAdult):
			response.BadRequest(w, "TUTOR_MUST_BE_ADULT", "A tutor must be 18 years old or older")
		case errors.Is(err, appatient.ErrSelfTutor):
			response.BadRequest(w, "SELF_TUTOR", "A patient cannot be their own tutor")
		case errors.Is(err, appatient.ErrEmailTaken):
			response.Conflict(w, "EMAIL_TAKEN", "This email is already used by another user account")
		case errors.Is(err, appatient.ErrEmailUsedByAnotherPatient):
			response.Conflict(w, "EMAIL_USED_BY_ANOTHER_PATIENT", "This email is already registered as the clinical email of another patient")
		case errors.Is(err, appatient.ErrLoginEmailRequired):
			response.BadRequest(w, "LOGIN_EMAIL_REQUIRED", "login_email is required: the patient has no clinical email to use as fallback")
		case errors.Is(err, appatient.ErrMinorRequiresTutor):
			response.BadRequest(w, "MINOR_REQUIRES_TUTOR", "Patients under 18 must be linked to a registered adult tutor")
		case errors.Is(err, appatient.ErrDuplicateCompanyID):
			response.BadRequest(w, "DUPLICATE_COMPANY_ID", "The same company_id appears more than once")
		case errors.As(err, &invComp):
			response.BadRequest(w, "INVALID_COMPANY_ID", err.Error())
		default:
			response.InternalError(w)
		}
		return
	}
	response.Created(w, p)
}

// GET /api/v1/patients/{id}
// Returns the full patient detail: companies, tutor, wards, login status.
func (h *PatientHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid patient ID")
		return
	}
	detail, err := h.svc.GetDetail(r.Context(), id)
	if err != nil {
		response.NotFound(w, "PATIENT_NOT_FOUND", "Patient not found")
		return
	}
	response.OK(w, detail)
}

// PUT /api/v1/patients/{id}
func (h *PatientHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid patient ID")
		return
	}

	var req appatient.UpdatePatientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}

	claims := middleware.ClaimsFromContext(r.Context())
	p, err := h.svc.Update(r.Context(), id, req, claims.UserID)
	if err != nil {
		var invComp *appatient.ErrInvalidCompanies
		switch {
		case errors.Is(err, appatient.ErrPatientNotFound):
			response.NotFound(w, "PATIENT_NOT_FOUND", "Patient not found")
		case errors.Is(err, appatient.ErrTutorNotFound):
			response.BadRequest(w, "TUTOR_NOT_FOUND", "The specified tutor patient does not exist")
		case errors.Is(err, appatient.ErrTutorMustBeAdult):
			response.BadRequest(w, "TUTOR_MUST_BE_ADULT", "A tutor must be 18 years old or older")
		case errors.Is(err, appatient.ErrSelfTutor):
			response.BadRequest(w, "SELF_TUTOR", "A patient cannot be their own tutor")
		case errors.Is(err, appatient.ErrMinorRequiresTutor):
			response.BadRequest(w, "MINOR_REQUIRES_TUTOR", "Patients under 18 must be linked to a registered adult tutor")
		case errors.Is(err, appatient.ErrDuplicateCompanyID):
			response.BadRequest(w, "DUPLICATE_COMPANY_ID", "The same company_id appears more than once")
		case errors.As(err, &invComp):
			response.BadRequest(w, "INVALID_COMPANY_ID", err.Error())
		default:
			response.InternalError(w)
		}
		return
	}
	response.OK(w, p)
}

// DELETE /api/v1/patients/{id}
func (h *PatientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid patient ID")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		response.NotFound(w, "PATIENT_NOT_FOUND", "Patient not found")
		return
	}
	response.NoContent(w)
}

// GET /api/v1/patients/{id}/companies
func (h *PatientHandler) GetCompanies(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid patient ID")
		return
	}
	companies, err := h.svc.GetCompanies(r.Context(), id)
	if err != nil {
		response.NotFound(w, "PATIENT_NOT_FOUND", "Patient not found")
		return
	}
	response.OK(w, companies)
}

// GET /api/v1/patients/{id}/clinical-record
func (h *PatientHandler) GetClinicalRecord(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid patient ID")
		return
	}
	cr, err := h.svc.GetClinicalRecord(r.Context(), id)
	if err != nil {
		response.NotFound(w, "CLINICAL_RECORD_NOT_FOUND", "Clinical record not found")
		return
	}
	response.OK(w, cr)
}

// PUT /api/v1/patients/{id}/clinical-record
func (h *PatientHandler) UpdateClinicalRecord(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid patient ID")
		return
	}

	var req appatient.UpdateClinicalRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}

	claims := middleware.ClaimsFromContext(r.Context())
	cr, err := h.svc.UpdateClinicalRecord(r.Context(), id, req, claims.UserID)
	if err != nil {
		response.NotFound(w, "CLINICAL_RECORD_NOT_FOUND", "Clinical record not found")
		return
	}
	response.OK(w, cr)
}

// POST /api/v1/patients/{id}/login
// Creates a portal user account for the patient.
func (h *PatientHandler) EnableLogin(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid patient ID")
		return
	}

	var req appatient.EnableLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	// login_password is always required; login_email is optional — when omitted
	// the service will use the patient's clinical email as the auth email automatically.
	if req.LoginPassword == "" {
		response.BadRequest(w, "MISSING_FIELDS", "login_password is required")
		return
	}

	claims := middleware.ClaimsFromContext(r.Context())
	if err := h.svc.EnableLogin(r.Context(), id, req, claims.UserID); err != nil {
		switch {
		case errors.Is(err, appatient.ErrPatientNotFound):
			response.NotFound(w, "PATIENT_NOT_FOUND", "Patient not found")
		case errors.Is(err, appatient.ErrLoginExists):
			response.Conflict(w, "LOGIN_EXISTS", "This patient already has an active login")
		case errors.Is(err, appatient.ErrEmailTaken):
			response.Conflict(w, "EMAIL_TAKEN", "This email is already used by another user account")
		case errors.Is(err, appatient.ErrEmailUsedByAnotherPatient):
			response.Conflict(w, "EMAIL_USED_BY_ANOTHER_PATIENT", "This email is already registered as the clinical email of another patient")
		case errors.Is(err, appatient.ErrLoginEmailRequired):
			response.BadRequest(w, "LOGIN_EMAIL_REQUIRED", "login_email is required: the patient has no clinical email to use as fallback")
		default:
			response.InternalError(w)
		}
		return
	}
	response.NoContent(w)
}

// DELETE /api/v1/patients/{id}/login
// Deactivates the portal user account for the patient.
func (h *PatientHandler) DisableLogin(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid patient ID")
		return
	}
	if err := h.svc.DisableLogin(r.Context(), id); err != nil {
		switch err {
		case appatient.ErrPatientNotFound:
			response.NotFound(w, "PATIENT_NOT_FOUND", "Patient not found")
		case appatient.ErrNoLogin:
			response.BadRequest(w, "NO_LOGIN", "This patient does not have an active login")
		default:
			response.InternalError(w)
		}
		return
	}
	response.NoContent(w)
}

// GET /api/v1/patients/{id}/wards
// Returns the list of minor patients for whom this patient is the registered tutor.
func (h *PatientHandler) GetWards(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid patient ID")
		return
	}
	wards, err := h.svc.GetWards(r.Context(), id)
	if err != nil {
		response.NotFound(w, "PATIENT_NOT_FOUND", "Patient not found")
		return
	}
	response.OK(w, wards)
}

// GET /api/v1/patients/{id}/login
// Returns the current portal login info for a patient (auth email, roles, active state).
// Used by the edit form to display existing login credentials.
func (h *PatientHandler) GetLoginInfo(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid patient ID")
		return
	}
	info, err := h.svc.GetLoginInfo(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, appatient.ErrPatientNotFound):
			response.NotFound(w, "PATIENT_NOT_FOUND", "Patient not found")
		case errors.Is(err, appatient.ErrNoLogin):
			response.NotFound(w, "NO_LOGIN", "This patient does not have an active login")
		default:
			response.InternalError(w)
		}
		return
	}
	response.OK(w, info)
}
