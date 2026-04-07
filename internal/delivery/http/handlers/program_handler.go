package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appprogram "amaur/api/internal/application/program"
	"amaur/api/internal/delivery/http/middleware"
	"amaur/api/internal/delivery/http/response"
	"amaur/api/pkg/pagination"

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

	claims := middleware.ClaimsFromContext(r.Context())
	if middleware.IsCompanyScopedRole(claims) {
		if claims.CompanyID == nil {
			response.Forbidden(w, "Missing company scope for current user")
			return
		}
		companyID = claims.CompanyID.String()
	}

	items, total, err := h.svc.ListPrograms(r.Context(), companyID, contractID, status, dateFrom, dateTo, p.Limit, p.Offset)
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
