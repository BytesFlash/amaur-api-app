package servicetype

import (
	"context"
	"errors"

	"amaur/api/internal/domain/servicetype"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("service type not found")

type CreateServiceTypeRequest struct {
	Name                   string   `json:"name"`
	Category               *string  `json:"category"`
	Description            *string  `json:"description"`
	DefaultDurationMinutes *int     `json:"default_duration_minutes"`
	IsGroupService         bool     `json:"is_group_service"`
	RequiresClinicalRecord bool     `json:"requires_clinical_record"`
	SpecialtyCodes         []string `json:"specialty_codes"`
}

type UpdateServiceTypeRequest struct {
	Name                   *string   `json:"name"`
	Category               *string   `json:"category"`
	Description            *string   `json:"description"`
	DefaultDurationMinutes *int      `json:"default_duration_minutes"`
	IsGroupService         *bool     `json:"is_group_service"`
	RequiresClinicalRecord *bool     `json:"requires_clinical_record"`
	IsActive               *bool     `json:"is_active"`
	SpecialtyCodes         *[]string `json:"specialty_codes"`
}

type Service struct {
	repo servicetype.Repository
}

func NewService(repo servicetype.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, activeOnly bool) ([]*servicetype.ServiceType, error) {
	return s.repo.List(ctx, activeOnly)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*servicetype.ServiceType, error) {
	st, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return st, nil
}

func (s *Service) Create(ctx context.Context, req CreateServiceTypeRequest) (*servicetype.ServiceType, error) {
	st := &servicetype.ServiceType{
		Name:                   req.Name,
		Category:               req.Category,
		Description:            req.Description,
		DefaultDurationMinutes: req.DefaultDurationMinutes,
		IsGroupService:         req.IsGroupService,
		RequiresClinicalRecord: req.RequiresClinicalRecord,
		IsActive:               true,
	}
	if err := s.repo.Create(ctx, st); err != nil {
		return nil, err
	}
	if len(req.SpecialtyCodes) > 0 {
		if err := s.repo.SetSpecialties(ctx, st.ID, req.SpecialtyCodes); err != nil {
			return nil, err
		}
	}
	return s.repo.FindByID(ctx, st.ID)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateServiceTypeRequest) (*servicetype.ServiceType, error) {
	st, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	if req.Name != nil {
		st.Name = *req.Name
	}
	if req.Category != nil {
		st.Category = req.Category
	}
	if req.Description != nil {
		st.Description = req.Description
	}
	if req.DefaultDurationMinutes != nil {
		st.DefaultDurationMinutes = req.DefaultDurationMinutes
	}
	if req.IsGroupService != nil {
		st.IsGroupService = *req.IsGroupService
	}
	if req.RequiresClinicalRecord != nil {
		st.RequiresClinicalRecord = *req.RequiresClinicalRecord
	}
	if req.IsActive != nil {
		st.IsActive = *req.IsActive
	}
	if err := s.repo.Update(ctx, st); err != nil {
		return nil, err
	}
	if req.SpecialtyCodes != nil {
		if err := s.repo.SetSpecialties(ctx, id, *req.SpecialtyCodes); err != nil {
			return nil, err
		}
	}
	return s.repo.FindByID(ctx, id)
}
