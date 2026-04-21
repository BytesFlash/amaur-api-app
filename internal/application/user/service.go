package user

import (
	"context"
	"database/sql"
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
	ErrSeedEmailRequired    = errors.New("seed admin email is required")
	ErrSeedPasswordRequired = errors.New("seed admin password is required")
)

type Service struct {
	repo       *postgres.UserRepository
	workerRepo worker.Repository
}

type EnsureSuperAdminRequest struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

type EnsureSuperAdminResult struct {
	UserID        uuid.UUID
	Email         string
	Created       bool
	RoleAssigned  bool
	AlreadyActive bool
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
	if err := validateRoleScopes(roleNames, req.CompanyID, req.PatientID); err != nil {
		return nil, err
	}

	existing, _ := s.repo.FindByEmail(ctx, req.Email)
	if existing != nil {
		return nil, ErrEmailTaken
	}
	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, err
	}
	var createdByRef *uuid.UUID
	if createdBy != uuid.Nil {
		createdByRef = &createdBy
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
		CreatedBy:    createdByRef,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	for _, roleID := range req.RoleIDs {
		if err := s.repo.AssignRole(ctx, u.ID, roleID, createdBy); err != nil {
			return nil, err
		}
	}
	if hasRole(roleNames, "professional") {
		if err := s.ensureProfessionalProfile(ctx, u, req, createdBy); err != nil {
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
	roleNames, err := s.repo.GetRoleNamesByIDs(ctx, req.RoleIDs)
	if err != nil {
		return err
	}
	if err := validateRoleScopes(roleNames, u.CompanyID, u.PatientID); err != nil {
		return err
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
	if hasRole(roleNames, "professional") {
		if err := s.ensureProfessionalProfile(ctx, u, CreateUserRequest{}, assignedBy); err != nil {
			return err
		}
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

func (s *Service) SyncProfessionalProfiles(ctx context.Context, createdBy uuid.UUID) (int, error) {
	users, err := s.repo.ListProfessionalsMissingWorker(ctx)
	if err != nil {
		return 0, err
	}
	synced := 0
	for _, u := range users {
		if err := s.ensureProfessionalProfile(ctx, u, CreateUserRequest{}, createdBy); err != nil {
			return synced, err
		}
		synced++
	}
	return synced, nil
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

func (s *Service) EnsureSuperAdmin(
	ctx context.Context,
	req EnsureSuperAdminRequest,
	createdBy uuid.UUID,
) (*EnsureSuperAdminResult, error) {
	email := normalizeEmail(req.Email)
	if email == "" {
		return nil, ErrSeedEmailRequired
	}
	passwordValue := strings.TrimSpace(req.Password)
	if passwordValue == "" {
		return nil, ErrSeedPasswordRequired
	}

	firstName := strings.TrimSpace(req.FirstName)
	if firstName == "" {
		firstName = "Super"
	}
	lastName := strings.TrimSpace(req.LastName)
	if lastName == "" {
		lastName = "Admin"
	}

	superAdminRoleID, err := s.GetRoleIDByName(ctx, "super_admin")
	if err != nil {
		return nil, err
	}

	result := &EnsureSuperAdminResult{
		Email: email,
	}

	existing, err := s.repo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if existing == nil {
		dto, createErr := s.Create(ctx, CreateUserRequest{
			Email:     email,
			Password:  passwordValue,
			FirstName: firstName,
			LastName:  lastName,
			RoleIDs:   []uuid.UUID{superAdminRoleID},
		}, createdBy)
		if createErr != nil && !errors.Is(createErr, ErrEmailTaken) {
			return nil, createErr
		}
		if createErr == nil && dto != nil {
			result.Created = true
		}

		existing, err = s.repo.FindByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
	}

	result.UserID = existing.ID
	result.AlreadyActive = existing.IsActive

	roleNames, err := s.repo.GetRoleNames(ctx, existing.ID)
	if err != nil {
		return nil, err
	}
	if !hasRole(roleNames, "super_admin") {
		if err := s.repo.AssignRole(ctx, existing.ID, superAdminRoleID, createdBy); err != nil {
			return nil, err
		}
		result.RoleAssigned = true
	}

	return result, nil
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
	var createdByRef *uuid.UUID
	if createdBy != uuid.Nil {
		createdByRef = &createdBy
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
		CreatedBy:         createdByRef,
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

func validateRoleScopes(roleNames []string, companyID, patientID *uuid.UUID) error {
	for _, rn := range roleNames {
		if (rn == "company_hr" || rn == "company_worker") && companyID == nil {
			return ErrCompanyScopeRequired
		}
		if rn == "company_worker" && patientID == nil {
			return ErrPatientScopeRequired
		}
	}
	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
