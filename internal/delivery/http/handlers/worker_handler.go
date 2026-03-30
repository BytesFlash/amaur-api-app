package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	appuser "amaur/api/internal/application/user"
	appworker "amaur/api/internal/application/worker"
	"amaur/api/internal/delivery/http/middleware"
	"amaur/api/internal/delivery/http/response"
	"amaur/api/pkg/pagination"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type WorkerHandler struct {
	svc *appworker.Service
}

func NewWorkerHandler(svc *appworker.Service) *WorkerHandler {
	return &WorkerHandler{svc: svc}
}

func (h *WorkerHandler) List(w http.ResponseWriter, r *http.Request) {
	p := pagination.FromRequest(r)
	search := r.URL.Query().Get("search")
	specialtyCode := r.URL.Query().Get("specialty_code")
	onlyActive := r.URL.Query().Get("active") != "false"
	workers, total, err := h.svc.List(r.Context(), search, specialtyCode, onlyActive, p.Limit, p.Offset)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Paginated(w, workers, pagination.NewMeta(p, total))
}

// ListSpecialties returns the active specialty catalog.
func (h *WorkerHandler) ListSpecialties(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListSpecialties(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, items)
}

// CreateSpecialty adds a new entry to the specialty catalog.
func (h *WorkerHandler) CreateSpecialty(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	if body.Code == "" || body.Name == "" {
		response.BadRequest(w, "REQUIRED", "code and name are required")
		return
	}
	item, err := h.svc.CreateSpecialty(r.Context(), body.Code, body.Name)
	if err != nil {
		response.Conflict(w, "DUPLICATE", "Ya existe una especialidad con ese código")
		return
	}
	response.Created(w, item)
}

// DeleteSpecialty removes a specialty from the catalog.
func (h *WorkerHandler) DeleteSpecialty(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if err := h.svc.DeleteSpecialty(r.Context(), code); err != nil {
		// FK violation means it's still in use
		response.BadRequest(w, "IN_USE", "La especialidad está en uso y no puede eliminarse")
		return
	}
	response.NoContent(w)
}

// SetWorkerSpecialties replaces the specialty list for a specific worker.
func (h *WorkerHandler) SetWorkerSpecialties(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid worker id")
		return
	}
	var body struct {
		Codes []string `json:"codes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	items, err := h.svc.SetWorkerSpecialties(r.Context(), id, body.Codes, claims.UserID)
	if err != nil {
		if errors.Is(err, appworker.ErrNotFound) {
			response.NotFound(w, "WORKER_NOT_FOUND", "Worker not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, items)
}

// GetWorkerCalendar returns per-day availability/booking summary for a given month.
func (h *WorkerHandler) GetWorkerCalendar(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid worker id")
		return
	}
	month := r.URL.Query().Get("month") // "YYYY-MM"
	days, err := h.svc.GetWorkerCalendar(r.Context(), id, month)
	if err != nil {
		if errors.Is(err, appworker.ErrNotFound) {
			response.NotFound(w, "WORKER_NOT_FOUND", "Worker not found")
			return
		}
		response.BadRequest(w, "INVALID_PARAM", err.Error())
		return
	}
	response.OK(w, days)
}

func (h *WorkerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req appworker.CreateWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	worker, err := h.svc.Create(r.Context(), req, claims.UserID)
	if err != nil {
		if errors.Is(err, appworker.ErrUserRequired) {
			response.BadRequest(w, "USER_REQUIRED", "user_id is required for worker profile")
			return
		}
		if errors.Is(err, appworker.ErrLoginEmailRequired) || errors.Is(err, appworker.ErrLoginPasswordRequired) {
			response.BadRequest(w, "VALIDATION_ERROR", err.Error())
			return
		}
		if errors.Is(err, appworker.ErrUserMustBeProfessional) {
			response.BadRequest(w, "ROLE_MISMATCH", "user must have professional role")
			return
		}
		if errors.Is(err, appworker.ErrUserAlreadyLinked) {
			response.BadRequest(w, "USER_ALREADY_LINKED", "user already linked to another worker")
			return
		}
		if errors.Is(err, appworker.ErrWorkerMustBeAdult) {
			response.BadRequest(w, "WORKER_MUST_BE_ADULT", err.Error())
			return
		}
		if errors.Is(err, appworker.ErrEmailUsedByAnotherUser) {
			response.Conflict(w, "EMAIL_IN_USE", err.Error())
			return
		}
		if errors.Is(err, appuser.ErrEmailTaken) {
			response.Conflict(w, "EMAIL_TAKEN", "Email already in use")
			return
		}
		response.InternalError(w)
		return
	}
	response.Created(w, worker)
}

func (h *WorkerHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid worker id")
		return
	}
	worker, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.NotFound(w, "WORKER_NOT_FOUND", "Worker not found")
		return
	}
	response.OK(w, worker)
}

func (h *WorkerHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid worker id")
		return
	}
	var req appworker.UpdateWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	worker, err := h.svc.Update(r.Context(), id, req, claims.UserID)
	if err != nil {
		if errors.Is(err, appworker.ErrNotFound) {
			response.NotFound(w, "WORKER_NOT_FOUND", "Worker not found")
			return
		}
		if errors.Is(err, appworker.ErrWorkerMustBeAdult) {
			response.BadRequest(w, "WORKER_MUST_BE_ADULT", err.Error())
			return
		}
		if errors.Is(err, appworker.ErrEmailUsedByAnotherUser) {
			response.Conflict(w, "EMAIL_IN_USE", err.Error())
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, worker)
}

func (h *WorkerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid worker id")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, appworker.ErrNotFound) {
			response.NotFound(w, "WORKER_NOT_FOUND", "Worker not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.NoContent(w)
}

// GetAvailabilityRules returns the weekly schedule rules for a worker.
func (h *WorkerHandler) GetAvailabilityRules(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid worker id")
		return
	}
	rules, err := h.svc.GetAvailabilityRules(r.Context(), id)
	if err != nil {
		if errors.Is(err, appworker.ErrNotFound) {
			response.NotFound(w, "WORKER_NOT_FOUND", "Worker not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, rules)
}

// SetAvailabilityRules replaces the weekly schedule rules for a worker.
func (h *WorkerHandler) SetAvailabilityRules(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid worker id")
		return
	}
	var body struct {
		Rules []appworker.AvailabilityRuleInput `json:"rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	rules, err := h.svc.SetAvailabilityRules(r.Context(), id, body.Rules, claims.UserID)
	if err != nil {
		if errors.Is(err, appworker.ErrNotFound) {
			response.NotFound(w, "WORKER_NOT_FOUND", "Worker not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, rules)
}

// GetWorkerSlots returns bookable time slots for a worker in a given week.
// Query params: week_start=YYYY-MM-DD (Monday), duration_minutes=60
func (h *WorkerHandler) GetWorkerSlots(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid worker id")
		return
	}
	weekStart := r.URL.Query().Get("week_start")
	durationStr := r.URL.Query().Get("duration_minutes")
	duration := 60
	if durationStr != "" {
		if _, err := fmt.Sscanf(durationStr, "%d", &duration); err != nil || duration < 1 {
			duration = 60
		}
	}
	slots, err := h.svc.GetWorkerSlots(r.Context(), id, weekStart, duration)
	if err != nil {
		if errors.Is(err, appworker.ErrNotFound) {
			response.NotFound(w, "WORKER_NOT_FOUND", "Worker not found")
			return
		}
		response.BadRequest(w, "INVALID_WEEK", err.Error())
		return
	}
	response.OK(w, slots)
}
