package contract

import (
	"context"
	"errors"
	"time"

	"amaur/api/internal/domain/contract"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("contract not found")

type CreateContractRequest struct {
	CompanyID         uuid.UUID `json:"company_id"`
	Name              string    `json:"name"`
	ContractType      *string   `json:"contract_type"`
	Status            string    `json:"status"`
	StartDate         string    `json:"start_date"`
	EndDate           *string   `json:"end_date"`
	RenewalDate       *string   `json:"renewal_date"`
	ValueCLP          *float64  `json:"value_clp"`
	BillingCycle      *string   `json:"billing_cycle"`
	Notes             *string   `json:"notes"`
	SignedDocumentURL *string   `json:"signed_document_url"`
}

type UpdateContractRequest struct {
	Name              *string  `json:"name"`
	ContractType      *string  `json:"contract_type"`
	Status            *string  `json:"status"`
	StartDate         *string  `json:"start_date"`
	EndDate           *string  `json:"end_date"`
	RenewalDate       *string  `json:"renewal_date"`
	ValueCLP          *float64 `json:"value_clp"`
	BillingCycle      *string  `json:"billing_cycle"`
	Notes             *string  `json:"notes"`
	SignedDocumentURL *string  `json:"signed_document_url"`
}

type Service struct {
	repo contract.Repository
}

func NewService(repo contract.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateContractRequest) (*contract.Contract, error) {
	if req.Status == "" {
		req.Status = "draft"
	}
	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return nil, errors.New("start_date inválido, use formato YYYY-MM-DD")
	}
	c := &contract.Contract{
		CompanyID:         req.CompanyID,
		Name:              req.Name,
		ContractType:      req.ContractType,
		Status:            req.Status,
		StartDate:         startDate,
		EndDate:           parseDatePtr(req.EndDate),
		RenewalDate:       parseDatePtr(req.RenewalDate),
		ValueCLP:          req.ValueCLP,
		BillingCycle:      req.BillingCycle,
		Notes:             req.Notes,
		SignedDocumentURL: req.SignedDocumentURL,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*contract.Contract, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return c, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateContractRequest) (*contract.Contract, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	if req.Name != nil {
		c.Name = *req.Name
	}
	if req.ContractType != nil {
		c.ContractType = req.ContractType
	}
	if req.Status != nil {
		c.Status = *req.Status
	}
	if req.StartDate != nil {
		t, err := parseDate(*req.StartDate)
		if err == nil {
			c.StartDate = t
		}
	}
	if req.EndDate != nil {
		c.EndDate = parseDatePtr(req.EndDate)
	}
	if req.RenewalDate != nil {
		c.RenewalDate = parseDatePtr(req.RenewalDate)
	}
	if req.ValueCLP != nil {
		c.ValueCLP = req.ValueCLP
	}
	if req.BillingCycle != nil {
		c.BillingCycle = req.BillingCycle
	}
	if req.Notes != nil {
		c.Notes = req.Notes
	}
	if req.SignedDocumentURL != nil {
		c.SignedDocumentURL = req.SignedDocumentURL
	}
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return ErrNotFound
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context, companyIDStr, status string, limit, offset int) ([]*contract.Contract, int64, error) {
	f := contract.Filter{}
	if companyIDStr != "" {
		if id, err := uuid.Parse(companyIDStr); err == nil {
			f.CompanyID = &id
		}
	}
	f.Status = status
	return s.repo.List(ctx, f, limit, offset)
}

func parseDate(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

func parseDatePtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := parseDate(*s)
	if err != nil {
		return nil
	}
	return &t
}

func (s *Service) ListServices(ctx context.Context, contractID uuid.UUID) ([]*contract.ContractService, error) {
	return s.repo.ListServices(ctx, contractID)
}
