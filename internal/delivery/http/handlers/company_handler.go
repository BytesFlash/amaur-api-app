package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appcompany "amaur/api/internal/application/company"
	appuser "amaur/api/internal/application/user"
	"amaur/api/internal/delivery/http/middleware"
	"amaur/api/internal/delivery/http/response"
	"amaur/api/internal/domain/company"
	"amaur/api/pkg/pagination"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CompanyHandler struct {
	svc *appcompany.Service
}

func NewCompanyHandler(svc *appcompany.Service) *CompanyHandler {
	return &CompanyHandler{svc: svc}
}

func (h *CompanyHandler) List(w http.ResponseWriter, r *http.Request) {
	p := pagination.FromRequest(r)
	f := company.Filter{
		Search:   r.URL.Query().Get("search"),
		Status:   r.URL.Query().Get("status"),
		Region:   r.URL.Query().Get("region"),
		Industry: r.URL.Query().Get("industry"),
	}
	if scopedCompanyID, ok := scopedCompanyID(w, r); !ok {
		return
	} else if scopedCompanyID != nil {
		f.ID = scopedCompanyID
	}
	companies, total, err := h.svc.List(r.Context(), f, p.Limit, p.Offset)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Paginated(w, companies, pagination.NewMeta(p, total))
}

func (h *CompanyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req appcompany.CreateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	c, err := h.svc.Create(r.Context(), req, claims.UserID)
	if err != nil {
		if errors.Is(err, appcompany.ErrRUTTaken) {
			response.Conflict(w, "RUT_TAKEN", "RUT already in use")
			return
		}
		if errors.Is(err, appcompany.ErrAdminEmailRequired) || errors.Is(err, appcompany.ErrAdminPasswordRequired) {
			response.BadRequest(w, "VALIDATION_ERROR", err.Error())
			return
		}
		if errors.Is(err, appuser.ErrEmailTaken) {
			response.Conflict(w, "EMAIL_TAKEN", "Email already in use")
			return
		}
		response.InternalError(w)
		return
	}
	response.Created(w, c)
}

func (h *CompanyHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid company id")
		return
	}
	if !ensureCompanyAccess(w, r, id) {
		return
	}
	c, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.NotFound(w, "COMPANY_NOT_FOUND", "Company not found")
		return
	}
	response.OK(w, c)
}

func (h *CompanyHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid company id")
		return
	}
	if !ensureCompanyAccess(w, r, id) {
		return
	}
	var req appcompany.UpdateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	c, err := h.svc.Update(r.Context(), id, req, claims.UserID)
	if err != nil {
		if errors.Is(err, appcompany.ErrNotFound) {
			response.NotFound(w, "COMPANY_NOT_FOUND", "Company not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, c)
}

func (h *CompanyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid company id")
		return
	}
	if !ensureCompanyAccess(w, r, id) {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, appcompany.ErrNotFound) {
			response.NotFound(w, "COMPANY_NOT_FOUND", "Company not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.NoContent(w)
}

func (h *CompanyHandler) ListBranches(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid company id")
		return
	}
	if !ensureCompanyAccess(w, r, id) {
		return
	}
	branches, err := h.svc.ListBranches(r.Context(), id)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, branches)
}

func (h *CompanyHandler) ListPatients(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid company id")
		return
	}
	if !ensureCompanyAccess(w, r, id) {
		return
	}
	p := pagination.FromRequest(r)
	patients, total, err := h.svc.ListPatients(r.Context(), id, p.Limit, p.Offset)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Paginated(w, patients, pagination.NewMeta(p, total))
}

func scopedCompanyID(w http.ResponseWriter, r *http.Request) (*uuid.UUID, bool) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !middleware.IsCompanyScopedRole(claims) {
		return nil, true
	}
	if claims.CompanyID == nil {
		response.Forbidden(w, "Missing company scope")
		return nil, false
	}
	return claims.CompanyID, true
}

func ensureCompanyAccess(w http.ResponseWriter, r *http.Request, companyID uuid.UUID) bool {
	scopedID, ok := scopedCompanyID(w, r)
	if !ok || scopedID == nil {
		return ok
	}
	if *scopedID != companyID {
		response.Forbidden(w, "You do not have access to this company")
		return false
	}
	return true
}
