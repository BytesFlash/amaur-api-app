package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appprogram "amaur/api/internal/application/program"
	"amaur/api/internal/delivery/http/middleware"
	"amaur/api/internal/delivery/http/response"
	"amaur/api/pkg/pagination"
	jwtpkg "amaur/api/pkg/jwt"
	"github.com/rs/zerolog/log"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ProgramHandler struct {
	svc *appprogram.Service
}

func NewProgramHandler(svc *appprogram.Service) *ProgramHandler {
	return &ProgramHandler{svc: svc}
}

func (h *ProgramHandler) List(w http.ResponseWriter, r *http.Request) {
	p := pagination.FromRequest(r)
	companyID := r.URL.Query().Get("company_id")
	contractID := r.URL.Query().Get("contract_id")
	status := r.URL.Query().Get("status")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")
	workerID := ""

	claims := middleware.ClaimsFromContext(r.Context())
	if middleware.IsCompanyScopedRole(claims) {
		if claims.CompanyID == nil {
			response.Forbidden(w, "Missing company scope for current user")
			return
		}
		companyID = claims.CompanyID.String()
	}
	// Professionals are scoped to programs where they are assigned as worker.
	// If they have the professional role but no worker_id linked, show nothing.
	if isProgramWorkerScoped(claims) {
		if claims.WorkerID == nil {
			response.Paginated(w, nil, pagination.NewMeta(p, 0))
			return
		}
		workerID = claims.WorkerID.String()
	}

	items, total, err := h.svc.ListPrograms(r.Context(), companyID, contractID, workerID, status, dateFrom, dateTo, p.Limit, p.Offset)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Paginated(w, items, pagination.NewMeta(p, total))
}

func (h *ProgramHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req appprogram.CreateProgramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}

	claims := middleware.ClaimsFromContext(r.Context())
	item, err := h.svc.CreateProgram(r.Context(), req, claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, appprogram.ErrContractNotFound):
			response.BadRequest(w, "CONTRACT_NOT_FOUND", err.Error())
		case errors.Is(err, appprogram.ErrContractCompanyMismatch):
			response.BadRequest(w, "CONTRACT_COMPANY_MISMATCH", err.Error())
		case errors.Is(err, appprogram.ErrInvalidDateRange):
			response.BadRequest(w, "INVALID_DATE_RANGE", err.Error())
		case errors.Is(err, appprogram.ErrOutsideContractRange):
			response.BadRequest(w, "OUTSIDE_CONTRACT_RANGE", err.Error())
		default:
			response.InternalError(w)
		}
		return
	}
	response.Created(w, item)
}

func (h *ProgramHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid program id")
		return
	}

	item, rules, err := h.svc.GetProgramByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, appprogram.ErrProgramNotFound) {
			response.NotFound(w, "PROGRAM_NOT_FOUND", "Program not found")
			return
		}
		response.InternalError(w)
		return
	}

	claims := middleware.ClaimsFromContext(r.Context())
	if middleware.IsCompanyScopedRole(claims) {
		if claims.CompanyID == nil || item.CompanyID != *claims.CompanyID {
			response.Forbidden(w, "You can only view programs from your company")
			return
		}
	}
	// Worker ownership check: professional can only view programs they are assigned to.
	// Uses IsWorkerLinkedToProgram so the check is consistent with the ListPrograms filter
	// (schedule rules OR agenda services), avoiding 403s when a worker was assigned
	// directly to individual services without being in the schedule rules.
	if isProgramWorkerScoped(claims) {
		if claims.WorkerID == nil {
			response.Forbidden(w, "Your account is not linked to a worker profile")
			return
		}
		assigned, linkErr := h.svc.IsWorkerLinkedToProgram(r.Context(), id, *claims.WorkerID)
		if linkErr != nil {
			response.InternalError(w)
			return
		}
		if !assigned {
			response.Forbidden(w, "You are not assigned to this program")
			return
		}
	}

	response.OK(w, map[string]any{
		"program": item,
		"rules":   rules,
	})
}

func (h *ProgramHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid program id")
		return
	}

	var req appprogram.UpdateProgramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}

	claims := middleware.ClaimsFromContext(r.Context())
	item, err := h.svc.UpdateProgram(r.Context(), id, req, claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, appprogram.ErrProgramNotFound):
			response.NotFound(w, "PROGRAM_NOT_FOUND", "Program not found")
		case errors.Is(err, appprogram.ErrContractNotFound):
			response.BadRequest(w, "CONTRACT_NOT_FOUND", err.Error())
		case errors.Is(err, appprogram.ErrInvalidDateRange):
			response.BadRequest(w, "INVALID_DATE_RANGE", err.Error())
		case errors.Is(err, appprogram.ErrOutsideContractRange):
			response.BadRequest(w, "OUTSIDE_CONTRACT_RANGE", err.Error())
		default:
			response.InternalError(w)
		}
		return
	}
	response.OK(w, item)
}

func (h *ProgramHandler) CreateAgendaService(w http.ResponseWriter, r *http.Request) {
	agendaID, err := uuid.Parse(chi.URLParam(r, "agendaId"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid agenda id")
		return
	}
	var req appprogram.CreateAgendaServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	req.AgendaID = agendaID

	item, err := h.svc.CreateAgendaService(r.Context(), req)
	if err != nil {
		if errors.Is(err, appprogram.ErrWorkerScheduleConflict) {
			response.Conflict(w, "WORKER_BUSY", err.Error())
			return
		}
		response.InternalError(w)
		return
	}
	response.Created(w, item)
}

func (h *ProgramHandler) UpsertParticipants(w http.ResponseWriter, r *http.Request) {
	agendaServiceID, err := uuid.Parse(chi.URLParam(r, "agendaServiceId"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid agenda service id")
		return
	}
	var payload struct {
		Participants []appprogram.ParticipantInput `json:"participants"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	if isProgramWorkerScoped(claims) {
		if !h.checkAgendaServiceWorkerOwnership(w, r, agendaServiceID) {
			return
		}
	}
	if err := h.svc.UpsertAgendaServiceParticipants(r.Context(), agendaServiceID, payload.Participants, claims.UserID); err != nil {
		if errors.Is(err, appprogram.ErrParticipantsOutsideCompany) {
			response.BadRequest(w, "PARTICIPANTS_OUTSIDE_COMPANY", err.Error())
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]any{"ok": true})
}

func (h *ProgramHandler) CompleteAgendaService(w http.ResponseWriter, r *http.Request) {
	agendaServiceID, err := uuid.Parse(chi.URLParam(r, "agendaServiceId"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid agenda service id")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	if isProgramWorkerScoped(claims) {
		if !h.checkAgendaServiceWorkerOwnership(w, r, agendaServiceID) {
			return
		}
	}
	if err := h.svc.CompleteAgendaService(r.Context(), agendaServiceID, claims.UserID); err != nil {
		switch {
		case errors.Is(err, appprogram.ErrAgendaServiceNotFound):
			response.NotFound(w, "AGENDA_SERVICE_NOT_FOUND", err.Error())
		case errors.Is(err, appprogram.ErrParticipantsOutsideCompany):
			response.BadRequest(w, "PARTICIPANTS_OUTSIDE_COMPANY", err.Error())
		case errors.Is(err, appprogram.ErrAgendaServiceWorkerRequired):
			response.BadRequest(w, "AGENDA_SERVICE_WORKER_REQUIRED", err.Error())
		default:
			response.InternalError(w)
		}
		return
	}
	response.OK(w, map[string]any{"ok": true})
}

// ListProgramAgendas returns agendas linked to a program, with their services.
func (h *ProgramHandler) ListProgramAgendas(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid program id")
		return
	}
	if !h.checkWorkerProgramAccess(w, r, id) {
		return
	}
	items, err := h.svc.GetProgramAgendas(r.Context(), id)
	if err != nil {
		if errors.Is(err, appprogram.ErrProgramNotFound) {
			response.NotFound(w, "PROGRAM_NOT_FOUND", "Program not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, items)
}

// GenerateAgendas generates agenda rows from program schedule rules.
func (h *ProgramHandler) GenerateAgendas(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid program id")
		return
	}
	if !h.checkWorkerProgramAccess(w, r, id) {
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	count, ids, err := h.svc.GenerateAgendas(r.Context(), id, claims.UserID)
	if err != nil {
		if errors.Is(err, appprogram.ErrProgramNotFound) {
			response.NotFound(w, "PROGRAM_NOT_FOUND", "Program not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.Created(w, map[string]any{"count": count, "agenda_ids": ids})
}

// ListAgendaServices lists services for a specific agenda.
func (h *ProgramHandler) ListAgendaServices(w http.ResponseWriter, r *http.Request) {
	agendaID, err := uuid.Parse(chi.URLParam(r, "agendaId"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid agenda id")
		return
	}
	items, err := h.svc.GetAgendaServices(r.Context(), agendaID)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, items)
}

// ListParticipants returns participants for an agenda service, enriched with patient names.
func (h *ProgramHandler) ListParticipants(w http.ResponseWriter, r *http.Request) {
	agendaServiceID, err := uuid.Parse(chi.URLParam(r, "agendaServiceId"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid agenda service id")
		return
	}
	items, err := h.svc.GetParticipantsDetail(r.Context(), agendaServiceID)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, items)
}

// ListPatientParticipation returns all group program sessions a patient has been enrolled in.
func (h *ProgramHandler) ListPatientParticipation(w http.ResponseWriter, r *http.Request) {
	patientID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid patient id")
		return
	}

	// A patient user may only fetch their own participations.
	claims := middleware.ClaimsFromContext(r.Context())
	if claims != nil && claims.PatientID != nil && !claims.HasPermission("patients:view") {
		if *claims.PatientID != patientID {
			response.Forbidden(w, "You can only view your own program participations")
			return
		}
	}

	items, err := h.svc.GetPatientParticipation(r.Context(), patientID)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, items)
}

// RegenerateAgendas clears pending agendas and regenerates from current rules.
func (h *ProgramHandler) RegenerateAgendas(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid program id")
		return
	}
	if !h.checkWorkerProgramAccess(w, r, id) {
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	count, ids, err := h.svc.RegenerateAgendas(r.Context(), id, claims.UserID)
	if err != nil {
		log.Error().Err(err).Str("program_id", id.String()).Msg("RegenerateAgendas failed")
		switch {
		case errors.Is(err, appprogram.ErrProgramNotFound):
			response.NotFound(w, "PROGRAM_NOT_FOUND", "Program not found")
		case errors.Is(err, appprogram.ErrContractNotFound):
			response.NotFound(w, "CONTRACT_NOT_FOUND", "Contract linked to this program was not found")
		default:
			response.InternalError(w)
		}
		return
	}
	response.Created(w, map[string]any{"count": count, "agenda_ids": ids})
}

// Delete removes a program permanently. Only allowed when the program has no
// completed sessions. Returns 409 if completed sessions exist.
func (h *ProgramHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid program id")
		return
	}
	if err := h.svc.DeleteProgram(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, appprogram.ErrProgramNotFound):
			response.NotFound(w, "PROGRAM_NOT_FOUND", "Program not found")
		case errors.Is(err, appprogram.ErrProgramHasCompletedSessions):
			response.Conflict(w, "HAS_COMPLETED_SESSIONS", err.Error())
		default:
			response.InternalError(w)
		}
		return
	}
	response.NoContent(w)
}

// isProgramWorkerScoped returns true when the user should only see programs they are
// assigned to as a worker. This covers the professional role (even if worker_id is nil
// in the JWT — that case must be handled explicitly by the caller) and any other
// non-super-admin user that carries a worker_id.
func isProgramWorkerScoped(claims *jwtpkg.Claims) bool {
	return claims.HasRole("professional") ||
		(!claims.HasRole("super_admin") && claims.WorkerID != nil)
}

// checkWorkerProgramAccess verifies that a worker-scoped user is assigned to the given
// program via schedule rules OR agenda services (consistent with the ListPrograms filter).
// It writes the appropriate HTTP error and returns false when access is denied;
// it returns true when the caller may proceed. Non-worker-scoped users always pass.
func (h *ProgramHandler) checkWorkerProgramAccess(w http.ResponseWriter, r *http.Request, programID uuid.UUID) bool {
	claims := middleware.ClaimsFromContext(r.Context())
	if !isProgramWorkerScoped(claims) {
		return true
	}
	if claims.WorkerID == nil {
		response.Forbidden(w, "Your account is not linked to a worker profile")
		return false
	}
	assigned, err := h.svc.IsWorkerLinkedToProgram(r.Context(), programID, *claims.WorkerID)
	if err != nil {
		response.InternalError(w)
		return false
	}
	if !assigned {
		response.Forbidden(w, "You are not assigned to this program")
		return false
	}
	return true
}

// checkAgendaServiceWorkerOwnership verifies that the worker-scoped caller is the
// worker assigned to the specific agenda service. This is the per-service ownership
// check for mutations (complete, upsert participants).
// It assumes isProgramWorkerScoped was already confirmed by the caller.
func (h *ProgramHandler) checkAgendaServiceWorkerOwnership(w http.ResponseWriter, r *http.Request, agendaServiceID uuid.UUID) bool {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims.WorkerID == nil {
		response.Forbidden(w, "Your account is not linked to a worker profile")
		return false
	}
	svc, err := h.svc.GetAgendaServiceByID(r.Context(), agendaServiceID)
	if err != nil {
		if errors.Is(err, appprogram.ErrAgendaServiceNotFound) {
			response.NotFound(w, "AGENDA_SERVICE_NOT_FOUND", "Agenda service not found")
		} else {
			response.InternalError(w)
		}
		return false
	}
	if svc.WorkerID == nil || *svc.WorkerID != *claims.WorkerID {
		response.Forbidden(w, "You are not the assigned professional for this service")
		return false
	}
	return true
}
