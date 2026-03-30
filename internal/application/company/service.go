package company

import (
	"context"
	"errors"
	"strings"
	"time"

	appuser "amaur/api/internal/application/user"
	"amaur/api/internal/domain/company"

	"github.com/google/uuid"
)

var (
	ErrNotFound              = errors.New("company not found")
	ErrRUTTaken              = errors.New("RUT already in use")
	ErrAdminEmailRequired    = errors.New("admin_email is required")
	ErrAdminPasswordRequired = errors.New("admin_password is required")
)

type CreateCompanyRequest struct {
	RUT             *string `json:"rut"`
	Name            string  `json:"name"          validate:"required"`
	FantasyName     *string `json:"fantasy_name"`
	Industry        *string `json:"industry"`
	SizeCategory    *string `json:"size_category"`
	ContactName     *string `json:"contact_name"`
	ContactEmail    *string `json:"contact_email"`
	ContactPhone    *string `json:"contact_phone"`
	BillingEmail    *string `json:"billing_email"`
	Address         *string `json:"address"`
	City            *string `json:"city"`
	Region          *string `json:"region"`
	Website         *string `json:"website"`
	AdminEmail      *string `json:"admin_email"`
	AdminPassword   *string `json:"admin_password"`
	AdminFirstName  *string `json:"admin_first_name"`
	AdminLastName   *string `json:"admin_last_name"`
	CommercialNotes *string `json:"commercial_notes"`
	LeadSource      *string `json:"lead_source"`
}

type UpdateCompanyRequest struct {
	RUT             *string `json:"rut"`
	Name            *string `json:"name"`
	FantasyName     *string `json:"fantasy_name"`
	Industry        *string `json:"industry"`
	SizeCategory    *string `json:"size_category"`
	ContactName     *string `json:"contact_name"`
	ContactEmail    *string `json:"contact_email"`
	ContactPhone    *string `json:"contact_phone"`
	BillingEmail    *string `json:"billing_email"`
	Address         *string `json:"address"`
	City            *string `json:"city"`
	Region          *string `json:"region"`
	Website         *string `json:"website"`
	Status          *string `json:"status"`
	CommercialNotes *string `json:"commercial_notes"`
	LeadSource      *string `json:"lead_source"`
}

type Service struct {
	repo    company.Repository
	userSvc *appuser.Service
}

func NewService(repo company.Repository, userSvc *appuser.Service) *Service {
	return &Service{repo: repo, userSvc: userSvc}
}

func (s *Service) Create(ctx context.Context, req CreateCompanyRequest, createdBy uuid.UUID) (*company.Company, error) {
	if req.RUT != nil {
		existing, _ := s.repo.FindByRUT(ctx, *req.RUT)
		if existing != nil {
			return nil, ErrRUTTaken
		}
	}

	c := &company.Company{
		ID:              uuid.New(),
		RUT:             req.RUT,
		Name:            req.Name,
		FantasyName:     req.FantasyName,
		Industry:        req.Industry,
		ContactName:     req.ContactName,
		ContactEmail:    req.ContactEmail,
		ContactPhone:    req.ContactPhone,
		BillingEmail:    req.BillingEmail,
		Address:         req.Address,
		City:            req.City,
		Region:          req.Region,
		Website:         req.Website,
		Status:          company.StatusActive,
		CommercialNotes: req.CommercialNotes,
		LeadSource:      req.LeadSource,
		CreatedBy:       &createdBy,
	}
	if req.SizeCategory != nil {
		sc := company.SizeCategory(*req.SizeCategory)
		c.SizeCategory = &sc
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}

	adminEmail := ""
	if req.AdminEmail != nil {
		adminEmail = strings.TrimSpace(*req.AdminEmail)
	}
	if adminEmail == "" && req.ContactEmail != nil {
		adminEmail = strings.TrimSpace(*req.ContactEmail)
	}
	if adminEmail == "" {
		_ = s.repo.SoftDelete(ctx, c.ID)
		return nil, ErrAdminEmailRequired
	}
	if req.AdminPassword == nil || strings.TrimSpace(*req.AdminPassword) == "" {
		_ = s.repo.SoftDelete(ctx, c.ID)
		return nil, ErrAdminPasswordRequired
	}

	roleID, err := s.userSvc.GetRoleIDByName(ctx, "company_hr")
	if err != nil {
		_ = s.repo.SoftDelete(ctx, c.ID)
		return nil, err
	}
	firstName, lastName := resolveAdminNames(req)
	_, err = s.userSvc.Create(ctx, appuser.CreateUserRequest{
		Email:     adminEmail,
		Password:  *req.AdminPassword,
		FirstName: firstName,
		LastName:  lastName,
		RoleIDs:   []uuid.UUID{roleID},
		CompanyID: &c.ID,
	}, createdBy)
	if err != nil {
		_ = s.repo.SoftDelete(ctx, c.ID)
		return nil, err
	}

	return c, nil
}

func resolveAdminNames(req CreateCompanyRequest) (string, string) {
	firstName := "Admin"
	lastName := "Empresa"
	if req.AdminFirstName != nil && strings.TrimSpace(*req.AdminFirstName) != "" {
		firstName = strings.TrimSpace(*req.AdminFirstName)
	}
	if req.AdminLastName != nil && strings.TrimSpace(*req.AdminLastName) != "" {
		lastName = strings.TrimSpace(*req.AdminLastName)
	}
	if (req.AdminFirstName == nil || strings.TrimSpace(*req.AdminFirstName) == "") && req.ContactName != nil {
		parts := strings.Fields(strings.TrimSpace(*req.ContactName))
		if len(parts) > 0 {
			firstName = parts[0]
		}
		if len(parts) > 1 {
			lastName = strings.Join(parts[1:], " ")
		}
	}
	return firstName, lastName
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*company.Company, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return c, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateCompanyRequest, updatedBy uuid.UUID) (*company.Company, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	now := time.Now()
	if req.Name != nil {
		c.Name = *req.Name
	}
	if req.FantasyName != nil {
		c.FantasyName = req.FantasyName
	}
	if req.RUT != nil {
		c.RUT = req.RUT
	}
	if req.Industry != nil {
		c.Industry = req.Industry
	}
	if req.ContactName != nil {
		c.ContactName = req.ContactName
	}
	if req.ContactEmail != nil {
		c.ContactEmail = req.ContactEmail
	}
	if req.ContactPhone != nil {
		c.ContactPhone = req.ContactPhone
	}
	if req.BillingEmail != nil {
		c.BillingEmail = req.BillingEmail
	}
	if req.Address != nil {
		c.Address = req.Address
	}
	if req.City != nil {
		c.City = req.City
	}
	if req.Region != nil {
		c.Region = req.Region
	}
	if req.Website != nil {
		c.Website = req.Website
	}
	if req.CommercialNotes != nil {
		c.CommercialNotes = req.CommercialNotes
	}
	if req.LeadSource != nil {
		c.LeadSource = req.LeadSource
	}
	if req.Status != nil {
		c.Status = company.CompanyStatus(*req.Status)
	}
	if req.SizeCategory != nil {
		sc := company.SizeCategory(*req.SizeCategory)
		c.SizeCategory = &sc
	}
	c.UpdatedAt = &now
	c.UpdatedBy = &updatedBy
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return ErrNotFound
	}
	return s.repo.SoftDelete(ctx, id)
}

func (s *Service) List(ctx context.Context, f company.Filter, limit, offset int) ([]*company.Company, int64, error) {
	return s.repo.List(ctx, f, limit, offset)
}

func (s *Service) CreateBranch(ctx context.Context, companyID uuid.UUID, name string, createdBy uuid.UUID) (*company.Branch, error) {
	b := &company.Branch{
		ID:        uuid.New(),
		CompanyID: companyID,
		Name:      name,
		IsMain:    false,
		IsActive:  true,
	}
	if err := s.repo.CreateBranch(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Service) ListBranches(ctx context.Context, companyID uuid.UUID) ([]*company.Branch, error) {
	return s.repo.ListBranches(ctx, companyID)
}

func (s *Service) ListPatients(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]*company.PatientSummary, int64, error) {
	return s.repo.ListPatients(ctx, companyID, limit, offset)
}
