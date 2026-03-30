package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"amaur/api/internal/domain/user"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	u := &user.User{}
	err := r.db.GetContext(ctx, u,
		`SELECT id, email, company_id, patient_id, password_hash, first_name, last_name, is_active,
                last_login_at, failed_attempts, locked_until, created_at, updated_at, created_by
         FROM users WHERE id = $1 AND deleted_at IS NULL`, id)
	return u, err
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	err := r.db.GetContext(ctx, &u,
		`SELECT id, email, company_id, patient_id, password_hash, first_name, last_name, is_active,
                last_login_at, failed_attempts, locked_until, created_at, updated_at, created_by
         FROM users WHERE LOWER(email) = LOWER(TRIM($1)) AND deleted_at IS NULL`, email)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, email, company_id, patient_id, password_hash, first_name, last_name, is_active, created_at, created_by)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		u.ID, u.Email, u.CompanyID, u.PatientID, u.PasswordHash, u.FirstName, u.LastName, u.IsActive, u.CreatedAt, u.CreatedBy)
	return err
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET email=$1, company_id=$2, patient_id=$3, first_name=$4, last_name=$5, is_active=$6, updated_at=$7 WHERE id=$8`,
		u.Email, u.CompanyID, u.PatientID, u.FirstName, u.LastName, u.IsActive, u.UpdatedAt, u.ID)
	return err
}

func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *UserRepository) ListActive(ctx context.Context, limit, offset int) ([]*user.User, int64, error) {
	var items []*user.User
	err := r.db.SelectContext(ctx, &items,
		`SELECT id, email, company_id, patient_id, first_name, last_name, is_active, created_at, updated_at
         FROM users WHERE is_active = true AND deleted_at IS NULL
         ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	var total int64
	_ = r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM users WHERE is_active = true AND deleted_at IS NULL`)
	return items, total, nil
}

func (r *UserRepository) GetRoleNames(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var roles []string
	err := r.db.SelectContext(ctx, &roles,
		`SELECT ro.name FROM roles ro
         JOIN user_roles ur ON ur.role_id = ro.id
         WHERE ur.user_id = $1`, userID)
	return roles, err
}

func (r *UserRepository) GetPermissionKeys(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var perms []string
	err := r.db.SelectContext(ctx, &perms,
		`SELECT DISTINCT p.module || ':' || p.action
         FROM permissions p
         JOIN role_permissions rp ON rp.permission_id = p.id
         JOIN user_roles ur ON ur.role_id = rp.role_id
         WHERE ur.user_id = $1`, userID)
	return perms, err
}

func (r *UserRepository) AssignRole(ctx context.Context, userID, roleID uuid.UUID, assignedBy uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_roles (user_id, role_id, assigned_at, assigned_by)
         VALUES ($1, $2, NOW(), $3) ON CONFLICT DO NOTHING`,
		userID, roleID, assignedBy)
	return err
}

func (r *UserRepository) RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM user_roles WHERE user_id=$1 AND role_id=$2`, userID, roleID)
	return err
}

func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET last_login_at = NOW() WHERE id = $1`, userID)
	return err
}

func (r *UserRepository) IncrementFailedAttempts(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET failed_attempts = failed_attempts + 1 WHERE id = $1`, userID)
	return err
}

func (r *UserRepository) ResetFailedAttempts(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET failed_attempts = 0, locked_until = NULL WHERE id = $1`, userID)
	return err
}

func (r *UserRepository) LockUntil(ctx context.Context, userID uuid.UUID, until string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET locked_until = $1 WHERE id = $2`, until, userID)
	return err
}

func (r *UserRepository) StoreRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
         VALUES ($1, $2, $3, $4, $5)`,
		uuid.New(), userID, tokenHash, expiresAt, time.Now())
	return err
}

func (r *UserRepository) FindRefreshToken(ctx context.Context, tokenHash string) (*user.RefreshToken, error) {
	rt := &user.RefreshToken{}
	err := r.db.GetContext(ctx, rt,
		`SELECT id, user_id, token_hash, expires_at, revoked_at
         FROM refresh_tokens
         WHERE token_hash = $1 AND expires_at > NOW()`, tokenHash)
	return rt, err
}

func (r *UserRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1`, tokenHash)
	return err
}

func (r *UserRepository) RevokeAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}

// UpdateWithPassword also updates the password_hash field.
func (r *UserRepository) UpdateWithPassword(ctx context.Context, u *user.User) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET email=$1, company_id=$2, patient_id=$3, first_name=$4, last_name=$5, is_active=$6,
         password_hash=$7, updated_at=NOW() WHERE id=$8`,
		u.Email, u.CompanyID, u.PatientID, u.FirstName, u.LastName, u.IsActive, u.PasswordHash, u.ID)
	return err
}

// ListAllRoles returns every role in the system.
func (r *UserRepository) ListAllRoles(ctx context.Context) ([]user.Role, error) {
	var roles []user.Role
	err := r.db.SelectContext(ctx, &roles,
		`SELECT id, name, description, is_system FROM roles ORDER BY name`)
	return roles, err
}

// GetUserRoleIDs returns the role UUIDs assigned to a user.
func (r *UserRepository) GetUserRoleIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.SelectContext(ctx, &ids,
		`SELECT role_id FROM user_roles WHERE user_id = $1`, userID)
	return ids, err
}

// GetRoleNamesByIDs returns role names from a list of role IDs.
func (r *UserRepository) GetRoleNamesByIDs(ctx context.Context, roleIDs []uuid.UUID) ([]string, error) {
	if len(roleIDs) == 0 {
		return []string{}, nil
	}
	var names []string
	err := r.db.SelectContext(ctx, &names,
		`SELECT name FROM roles WHERE id = ANY($1)`, pq.Array(roleIDs))
	return names, err
}

// FindByPatientID returns the active (non-deleted) user linked to the given
// patient, or an error if no such user exists.
func (r *UserRepository) FindByPatientID(ctx context.Context, patientID uuid.UUID) (*user.User, error) {
	u := &user.User{}
	err := r.db.GetContext(ctx, u,
		`SELECT id, email, company_id, patient_id, password_hash, first_name, last_name, is_active,
                last_login_at, failed_attempts, locked_until, created_at, updated_at, created_by
         FROM users
         WHERE patient_id = $1 AND deleted_at IS NULL
         LIMIT 1`, patientID)
	return u, err
}

// FindSoftDeletedByPatientID returns the most-recently soft-deleted user that
// was linked to patientID, or nil if no such record exists.
func (r *UserRepository) FindSoftDeletedByPatientID(ctx context.Context, patientID uuid.UUID) (*user.User, error) {
	u := &user.User{}
	err := r.db.GetContext(ctx, u,
		`SELECT id, email, company_id, patient_id, password_hash, first_name, last_name, is_active,
                last_login_at, failed_attempts, locked_until, created_at, updated_at, created_by
         FROM users
         WHERE patient_id = $1 AND deleted_at IS NOT NULL
         ORDER BY deleted_at DESC
         LIMIT 1`, patientID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

// Reactivate restores a soft-deleted user: clears deleted_at, sets is_active=true,
// and refreshes the email, password_hash, first_name and last_name fields.
// Call this when re-enabling a patient login that was previously disabled via SoftDelete.
func (r *UserRepository) Reactivate(ctx context.Context, u *user.User) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users
         SET deleted_at = NULL,
             is_active = true,
             failed_attempts = 0,
             locked_until = NULL,
             email = $1,
             password_hash = $2,
             first_name = $3,
             last_name = $4,
             updated_at = $5
         WHERE id = $6`,
		u.Email, u.PasswordHash, u.FirstName, u.LastName, time.Now(), u.ID)
	return err
}

// GetRoleIDByName returns the UUID of a role that matches the given name.
func (r *UserRepository) GetRoleIDByName(ctx context.Context, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.GetContext(ctx, &id,
		`SELECT id FROM roles WHERE name = $1 LIMIT 1`, name)
	return id, err
}
