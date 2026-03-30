package user

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	// FindByPatientID returns the active (non-deleted) user linked to the given patient, or nil.
	FindByPatientID(ctx context.Context, patientID uuid.UUID) (*User, error)
	// FindSoftDeletedByPatientID returns the most-recently soft-deleted user
	// that was linked to the given patient, or nil if none exists.
	// Used by enableLogin to reactivate a previous login instead of creating a duplicate.
	FindSoftDeletedByPatientID(ctx context.Context, patientID uuid.UUID) (*User, error)
	Create(ctx context.Context, u *User) error
	Update(ctx context.Context, u *User) error
	// Reactivate restores a soft-deleted user: clears deleted_at, sets is_active=true,
	// and refreshes email, password_hash, first_name, and last_name.
	// Use this instead of Create when re-enabling a patient login that was previously disabled.
	Reactivate(ctx context.Context, u *User) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	ListActive(ctx context.Context, limit, offset int) ([]*User, int64, error)

	// Roles & permissions
	GetRoleNames(ctx context.Context, userID uuid.UUID) ([]string, error)
	GetPermissionKeys(ctx context.Context, userID uuid.UUID) ([]string, error)
	AssignRole(ctx context.Context, userID, roleID uuid.UUID, assignedBy uuid.UUID) error
	RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error
	// GetRoleIDByName returns the UUID of a role by its name.
	GetRoleIDByName(ctx context.Context, name string) (uuid.UUID, error)

	// Auth
	UpdateLastLogin(ctx context.Context, userID uuid.UUID) error
	IncrementFailedAttempts(ctx context.Context, userID uuid.UUID) error
	ResetFailedAttempts(ctx context.Context, userID uuid.UUID) error
	LockUntil(ctx context.Context, userID uuid.UUID, until string) error

	// Refresh tokens
	StoreRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt string) error
	FindRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllRefreshTokens(ctx context.Context, userID uuid.UUID) error
}

type Role struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	IsSystem    bool      `db:"is_system"`
}

type RefreshToken struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	ExpiresAt string    `db:"expires_at"`
	RevokedAt *string   `db:"revoked_at"`
}
