package handlers

import (
"encoding/json"
"errors"
"net/http"

appcontract "amaur/api/internal/application/contract"
"amaur/api/internal/delivery/http/response"
"amaur/api/pkg/pagination"

"github.com/go-chi/chi/v5"
"github.com/google/uuid"
)

type ContractHandler struct {
svc *appcontract.Service
}

func NewContractHandler(svc *appcontract.Service) *ContractHandler {
return &ContractHandler{svc: svc}
}

func (h *ContractHandler) List(w http.ResponseWriter, r *http.Request) {
p := pagination.FromRequest(r)
companyID := r.URL.Query().Get("company_id")
status := r.URL.Query().Get("status")
contracts, total, err := h.svc.List(r.Context(), companyID, status, p.Limit, p.Offset)
if err != nil {
response.InternalError(w)
return
}
response.Paginated(w, contracts, pagination.NewMeta(p, total))
}

func (h *ContractHandler) Create(w http.ResponseWriter, r *http.Request) {
var req appcontract.CreateContractRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
response.BadRequest(w, "INVALID_BODY", "Invalid request body")
return
}
contract, err := h.svc.Create(r.Context(), req)
if err != nil {
response.InternalError(w)
return
}
response.Created(w, contract)
}

func (h *ContractHandler) GetByID(w http.ResponseWriter, r *http.Request) {
id, err := uuid.Parse(chi.URLParam(r, "id"))
if err != nil {
response.BadRequest(w, "INVALID_ID", "Invalid contract id")
return
}
contract, err := h.svc.GetByID(r.Context(), id)
if err != nil {
response.NotFound(w, "CONTRACT_NOT_FOUND", "Contract not found")
return
}
response.OK(w, contract)
}

func (h *ContractHandler) Update(w http.ResponseWriter, r *http.Request) {
id, err := uuid.Parse(chi.URLParam(r, "id"))
if err != nil {
response.BadRequest(w, "INVALID_ID", "Invalid contract id")
return
}
var req appcontract.UpdateContractRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
response.BadRequest(w, "INVALID_BODY", "Invalid request body")
return
}
contract, err := h.svc.Update(r.Context(), id, req)
if errors.Is(err, appcontract.ErrNotFound) {
response.NotFound(w, "CONTRACT_NOT_FOUND", "Contract not found")
return
}
if err != nil {
response.InternalError(w)
return
}
response.OK(w, contract)
}

func (h *ContractHandler) Delete(w http.ResponseWriter, r *http.Request) {
id, err := uuid.Parse(chi.URLParam(r, "id"))
if err != nil {
response.BadRequest(w, "INVALID_ID", "Invalid contract id")
return
}
if err := h.svc.Delete(r.Context(), id); err != nil {
if errors.Is(err, appcontract.ErrNotFound) {
response.NotFound(w, "CONTRACT_NOT_FOUND", "Contract not found")
return
}
response.InternalError(w)
return
}
response.NoContent(w)
}

func (h *ContractHandler) ListServices(w http.ResponseWriter, r *http.Request) {
id, err := uuid.Parse(chi.URLParam(r, "id"))
if err != nil {
response.BadRequest(w, "INVALID_ID", "Invalid contract id")
return
}
services, err := h.svc.ListServices(r.Context(), id)
if err != nil {
response.InternalError(w)
return
}
response.OK(w, services)
}