package user

import (
	"context"
	"errors"
	"strings"

	"amaur/api/internal/domain/user"
	"amaur/api/internal/domain/worker"
	"amaur/api/internal/infrastructure/postgres"
	"amaur/api/pkg/password"

	"github.com/google/uuid"
)

var (
	ErrEmailTaken           = errors.New("email already in use")
	ErrUserNotFound         = errors.New("user not found")
	ErrRoleRequired         = errors.New("at least one role is required")
	ErrRoleNotFound         = errors.New("role not found")
	ErrCompanyScopeRequired = errors.New("company_id is required for company-scoped roles")
	ErrPatientScopeRequired = errors.New("patient_id is required for company_worker role")
)

type Service struct {
	repo       *postgres.UserRepository
	workerRepo worker.Repository
}

func NewService(repo *postgres.UserRepository, workerRepo worker.Repository) *Service {
	return &Service{repo: repo, workerRepo: workerRepo}
}

func (s *Service) Create(ctx context.Context, req CreateUserRequest, createdBy uuid.UUID) (*UserDTO, error) {
	req.Email = normalizeEmail(req.Email)
	if len(req.RoleIDs) == 0 {
		return nil, ErrRoleRequired
	}
	roleNames, err := s.repo.GetRoleNamesByIDs(ctx, req.RoleIDs)
	if err != nil {
		return nil, err
	}
	for _, rn := range roleNames {
		if (rn == "company_hr" || rn == "company_worker") && req.CompanyID == nil {
			return nil, ErrCompanyScopeRequired
		}
	}

	existing, _ := s.repo.FindByEmail(ctx, req.Email)
	if existing != nil {
		return nil, ErrEmailTaken
	}
	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, err
	}
	u := &user.User{
		ID:           uuid.New(),
		Email:        req.Email,
		CompanyID:    req.CompanyID,
		PatientID:    req.PatientID,
		PasswordHash: hash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		IsActive:     true,
		CreatedBy:    &createdBy,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	for _, roleID := range req.RoleIDs {
		if err := s.repo.AssignRole(ctx, u.ID, roleID, createdBy); err != nil {
			return nil, err
		}
	}
	roles, _ := s.repo.GetRoleNames(ctx, u.ID)
	return toDTO(u, roles), nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*UserDTO, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	roles, _ := s.repo.GetRoleNames(ctx, id)
	return toDTO(u, roles), nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*UserDTO, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	if req.FirstName != nil {
		u.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		u.LastName = *req.LastName
	}
	if req.IsActive != nil {
		u.IsActive = *req.IsActive
	}
	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	roles, _ := s.repo.GetRoleNames(ctx, id)
	return toDTO(u, roles), nil
}

func (s *Service) ChangePassword(ctx context.Context, id uuid.UUID, req ChangePasswordRequest) error {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return ErrUserNotFound
	}
	hash, err := password.Hash(req.NewPassword)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	return s.repo.UpdateWithPassword(ctx, u)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return ErrUserNotFound
	}
	return s.repo.SoftDelete(ctx, id)
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]*UserDTO, int64, error) {
	users, total, err := s.repo.ListActive(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	dtos := make([]*UserDTO, 0, len(users))
	for _, u := range users {
		roles, _ := s.repo.GetRoleNames(ctx, u.ID)
		dtos = append(dtos, toDTO(u, roles))
	}
	return dtos, total, nil
}

func (s *Service) AssignRoles(ctx context.Context, userID uuid.UUID, req AssignRolesRequest, assignedBy uuid.UUID) error {
	if len(req.RoleIDs) == 0 {
		return ErrRoleRequired
	}
	u, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}
	currentRoles, _ := s.repo.GetUserRoleIDs(ctx, userID)
	for _, r := range currentRoles {
		_ = s.repo.RevokeRole(ctx, userID, r)
	}
	for _, roleID := range req.RoleIDs {
		if err := s.repo.AssignRole(ctx, userID, roleID, assignedBy); err != nil {
			return err
		}
	}
	roleNames, err := s.repo.GetRoleNamesByIDs(ctx, req.RoleIDs)
	if err == nil && hasRole(roleNames, "professional") {
		_ = s.ensureProfessionalProfile(ctx, u, CreateUserRequest{}, assignedBy)
	}
	return nil
}

func (s *Service) ListRoles(ctx context.Context) ([]*RoleDTO, error) {
	roles, err := s.repo.ListAllRoles(ctx)
	if err != nil {
		return nil, err
	}
	dtos := make([]*RoleDTO, len(roles))
	for i, r := range roles {
		dtos[i] = &RoleDTO{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			IsSystem:    r.IsSystem,
		}
	}
	return dtos, nil
}

func (s *Service) GetRoleIDByName(ctx context.Context, roleName string) (uuid.UUID, error) {
	roles, err := s.repo.ListAllRoles(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	for _, r := range roles {
		if r.Name == roleName {
			return r.ID, nil
		}
	}
	return uuid.Nil, ErrRoleNotFound
}

func toDTO(u *user.User, roles []string) *UserDTO {
	if roles == nil {
		roles = []string{}
	}
	return &UserDTO{
		ID:          u.ID,
		Email:       u.Email,
		FirstName:   u.FirstName,
		LastName:    u.LastName,
		IsActive:    u.IsActive,
		LastLoginAt: u.LastLoginAt,
		Roles:       roles,
		CompanyID:   u.CompanyID,
		PatientID:   u.PatientID,
		CreatedAt:   u.CreatedAt,
	}
}

func (s *Service) ensureProfessionalProfile(ctx context.Context, u *user.User, req CreateUserRequest, createdBy uuid.UUID) error {
	if s.workerRepo == nil {
		return nil
	}
	if _, err := s.workerRepo.FindByUserID(ctx, u.ID); err == nil {
		return nil
	}
	email := req.Email
	if email == "" {
		email = u.Email
	}
	w := &worker.Worker{
		ID:                uuid.New(),
		UserID:            &u.ID,
		RUT:               req.RUT,
		FirstName:         u.FirstName,
		LastName:          u.LastName,
		Email:             &email,
		Phone:             req.Phone,
		RoleTitle:         req.RoleTitle,
		Specialty:         req.Specialty,
		IsActive:          true,
		AvailabilityNotes: req.AvailabilityNotes,
		CreatedBy:         &createdBy,
	}
	return s.workerRepo.Create(ctx, w)
}

func hasRole(roleNames []string, target string) bool {
	for _, rn := range roleNames {
		if rn == target {
			return true
		}
	}
	return false
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
