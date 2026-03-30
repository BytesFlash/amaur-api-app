package handlers

import (
	"encoding/json"
	"net/http"

	appst "amaur/api/internal/application/servicetype"
	"amaur/api/internal/delivery/http/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ServiceTypeHandler struct {
	svc *appst.Service
}

func NewServiceTypeHandler(svc *appst.Service) *ServiceTypeHandler {
	return &ServiceTypeHandler{svc: svc}
}

// GET /service-types
func (h *ServiceTypeHandler) List(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active") != "false"
	items, err := h.svc.List(r.Context(), activeOnly)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, items)
}

// POST /service-types
func (h *ServiceTypeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req appst.CreateServiceTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	st, err := h.svc.Create(r.Context(), req)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Created(w, st)
}

// PATCH /service-types/{id}
func (h *ServiceTypeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid service type id")
		return
	}
	var req appst.UpdateServiceTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	st, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		response.NotFound(w, "NOT_FOUND", "Service type not found")
		return
	}
	response.OK(w, st)
}
