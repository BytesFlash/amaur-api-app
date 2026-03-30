package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"amaur/api/internal/domain/user"
	jwtpkg "amaur/api/pkg/jwt"
	"amaur/api/pkg/password"
)

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrUserInactive        = errors.New("user account is disabled")
	ErrAccountLocked       = errors.New("account temporarily locked due to multiple failed attempts")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

const maxFailedAttempts = 5

type Service struct {
	userRepo user.Repository
	jwt      *jwtpkg.Manager
}

func NewService(userRepo user.Repository, jwt *jwtpkg.Manager) *Service {
	return &Service{userRepo: userRepo, jwt: jwt}
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	u, err := s.userRepo.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !u.IsActive {
		return nil, ErrUserInactive
	}

	if u.IsLocked() {
		return nil, ErrAccountLocked
	}

	if !password.Verify(req.Password, u.PasswordHash) {
		_ = s.userRepo.IncrementFailedAttempts(ctx, u.ID)
		if u.FailedAttempts+1 >= maxFailedAttempts {
			lockUntil := time.Now().Add(15 * time.Minute).Format(time.RFC3339)
			_ = s.userRepo.LockUntil(ctx, u.ID, lockUntil)
		}
		return nil, ErrInvalidCredentials
	}

	_ = s.userRepo.ResetFailedAttempts(ctx, u.ID)
	_ = s.userRepo.UpdateLastLogin(ctx, u.ID)

	roles, err := s.userRepo.GetRoleNames(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	perms, err := s.userRepo.GetPermissionKeys(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.jwt.GenerateAccessToken(u.ID, u.Email, u.CompanyID, u.PatientID, roles, perms)
	if err != nil {
		return nil, err
	}

	rawRefresh := s.jwt.GenerateRefreshToken()
	tokenHash := hashToken(rawRefresh)
	expiresAt := time.Now().Add(s.jwt.RefreshExpiry())

	if err := s.userRepo.StoreRefreshToken(ctx, u.ID, tokenHash, expiresAt.Format(time.RFC3339)); err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		User: UserDTO{
			ID:          u.ID,
			Email:       u.Email,
			CompanyID:   u.CompanyID,
			PatientID:   u.PatientID,
			FirstName:   u.FirstName,
			LastName:    u.LastName,
			Roles:       roles,
			Permissions: perms,
		},
	}, nil
}

func (s *Service) Refresh(ctx context.Context, req RefreshRequest) (*LoginResponse, error) {
	tokenHash := hashToken(req.RefreshToken)

	rt, err := s.userRepo.FindRefreshToken(ctx, tokenHash)
	if err != nil || rt.RevokedAt != nil {
		return nil, ErrInvalidRefreshToken
	}

	u, err := s.userRepo.FindByID(ctx, rt.UserID)
	if err != nil || !u.IsActive {
		return nil, ErrInvalidRefreshToken
	}

	// Rotate token
	_ = s.userRepo.RevokeRefreshToken(ctx, tokenHash)

	roles, _ := s.userRepo.GetRoleNames(ctx, u.ID)
	perms, _ := s.userRepo.GetPermissionKeys(ctx, u.ID)

	accessToken, err := s.jwt.GenerateAccessToken(u.ID, u.Email, u.CompanyID, u.PatientID, roles, perms)
	if err != nil {
		return nil, err
	}

	rawRefresh := s.jwt.GenerateRefreshToken()
	newHash := hashToken(rawRefresh)
	expiresAt := time.Now().Add(s.jwt.RefreshExpiry())
	_ = s.userRepo.StoreRefreshToken(ctx, u.ID, newHash, expiresAt.Format(time.RFC3339))

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		User: UserDTO{
			ID: u.ID, Email: u.Email,
			CompanyID: u.CompanyID,
			PatientID: u.PatientID,
			FirstName: u.FirstName, LastName: u.LastName,
			Roles: roles, Permissions: perms,
		},
	}, nil
}

func (s *Service) Logout(ctx context.Context, userID interface{ String() string }, rawRefreshToken string) error {
	tokenHash := hashToken(rawRefreshToken)
	return s.userRepo.RevokeRefreshToken(ctx, tokenHash)
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", h)
}
