package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"amaur/api/internal/domain/user"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	u := &user.User{}
	err := rawGet(ctx, r.db, u,
		`SELECT id, email, company_id, patient_id, password_hash, first_name, last_name, is_active,
                last_login_at, failed_attempts, locked_until, created_at, updated_at, created_by
         FROM users WHERE id = $1 AND deleted_at IS NULL`, id)
	return u, err
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	err := rawGet(ctx, r.db, &u,
		`SELECT id, email, company_id, patient_id, password_hash, first_name, last_name, is_active,
                last_login_at, failed_attempts, locked_until, created_at, updated_at, created_by
         FROM users WHERE LOWER(email) = LOWER(TRIM($1)) AND deleted_at IS NULL`, email)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	return rawExec(ctx, r.db,
		`INSERT INTO users (id, email, company_id, patient_id, password_hash, first_name, last_name, is_active, created_at, created_by)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		u.ID, u.Email, u.CompanyID, u.PatientID, u.PasswordHash, u.FirstName, u.LastName, u.IsActive, u.CreatedAt, u.CreatedBy)
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	return rawExec(ctx, r.db,
		`UPDATE users SET email=$1, company_id=$2, patient_id=$3, first_name=$4, last_name=$5, is_active=$6, updated_at=$7 WHERE id=$8`,
		u.Email, u.CompanyID, u.PatientID, u.FirstName, u.LastName, u.IsActive, u.UpdatedAt, u.ID)
}

func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return rawExec(ctx, r.db, `UPDATE users SET deleted_at = NOW() WHERE id = $1`, id)
}

func (r *UserRepository) ListActive(ctx context.Context, limit, offset int) ([]*user.User, int64, error) {
	var items []*user.User
	err := rawSelectPtr(ctx, r.db, &items,
		`SELECT id, email, company_id, patient_id, first_name, last_name, is_active, created_at, updated_at
         FROM users WHERE is_active = true AND deleted_at IS NULL
         ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	var countRow struct {
		Count int64 `gorm:"column:count"`
	}
	_ = rawGet(ctx, r.db, &countRow,
		`SELECT COUNT(*) AS count FROM users WHERE is_active = true AND deleted_at IS NULL`)
	return items, countRow.Count, nil
}

func (r *UserRepository) GetRoleNames(ctx context.Context, userID uuid.UUID) ([]string, error) {
	type row struct {
		Name string `gorm:"column:name"`
	}
	var rows []row
	err := rawSelect(ctx, r.db, &rows,
		`SELECT ro.name FROM roles ro
         JOIN user_roles ur ON ur.role_id = ro.id
         WHERE ur.user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Name)
	}
	return out, nil
}

func (r *UserRepository) GetPermissionKeys(ctx context.Context, userID uuid.UUID) ([]string, error) {
	type row struct {
		Key string `gorm:"column:key"`
	}
	var rows []row
	err := rawSelect(ctx, r.db, &rows,
		`SELECT DISTINCT p.module || ':' || p.action AS key
         FROM permissions p
         JOIN role_permissions rp ON rp.permission_id = p.id
         JOIN user_roles ur ON ur.role_id = rp.role_id
         WHERE ur.user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Key)
	}
	return out, nil
}

func (r *UserRepository) AssignRole(ctx context.Context, userID, roleID uuid.UUID, assignedBy uuid.UUID) error {
	return rawExec(ctx, r.db,
		`INSERT INTO user_roles (user_id, role_id, assigned_at, assigned_by)
         VALUES ($1, $2, NOW(), $3) ON CONFLICT DO NOTHING`,
		userID, roleID, assignedBy)
}

func (r *UserRepository) RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error {
	return rawExec(ctx, r.db, `DELETE FROM user_roles WHERE user_id=$1 AND role_id=$2`, userID, roleID)
}

func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	return rawExec(ctx, r.db, `UPDATE users SET last_login_at = NOW() WHERE id = $1`, userID)
}

func (r *UserRepository) IncrementFailedAttempts(ctx context.Context, userID uuid.UUID) error {
	return rawExec(ctx, r.db, `UPDATE users SET failed_attempts = failed_attempts + 1 WHERE id = $1`, userID)
}

func (r *UserRepository) ResetFailedAttempts(ctx context.Context, userID uuid.UUID) error {
	return rawExec(ctx, r.db, `UPDATE users SET failed_attempts = 0, locked_until = NULL WHERE id = $1`, userID)
}

func (r *UserRepository) LockUntil(ctx context.Context, userID uuid.UUID, until string) error {
	return rawExec(ctx, r.db, `UPDATE users SET locked_until = $1 WHERE id = $2`, until, userID)
}

func (r *UserRepository) StoreRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt string) error {
	return rawExec(ctx, r.db,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
         VALUES ($1, $2, $3, $4, $5)`,
		uuid.New(), userID, tokenHash, expiresAt, time.Now())
}

func (r *UserRepository) FindRefreshToken(ctx context.Context, tokenHash string) (*user.RefreshToken, error) {
	rt := &user.RefreshToken{}
	err := rawGet(ctx, r.db, rt,
		`SELECT id, user_id, token_hash, expires_at, revoked_at
         FROM refresh_tokens
         WHERE token_hash = $1 AND expires_at > NOW()`, tokenHash)
	return rt, err
}

func (r *UserRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	return rawExec(ctx, r.db, `UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1`, tokenHash)
}

func (r *UserRepository) RevokeAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	return rawExec(ctx, r.db, `UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
}

func (r *UserRepository) UpdateWithPassword(ctx context.Context, u *user.User) error {
	return rawExec(ctx, r.db,
		`UPDATE users SET email=$1, company_id=$2, patient_id=$3, first_name=$4, last_name=$5, is_active=$6,
         password_hash=$7, updated_at=NOW() WHERE id=$8`,
		u.Email, u.CompanyID, u.PatientID, u.FirstName, u.LastName, u.IsActive, u.PasswordHash, u.ID)
}

func (r *UserRepository) ListAllRoles(ctx context.Context) ([]user.Role, error) {
	var roles []user.Role
	err := rawSelect(ctx, r.db, &roles, `SELECT id, name, description, is_system FROM roles ORDER BY name`)
	return roles, err
}

func (r *UserRepository) GetUserRoleIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	type row struct {
		RoleID uuid.UUID `gorm:"column:role_id"`
	}
	var rows []row
	err := rawSelect(ctx, r.db, &rows, `SELECT role_id FROM user_roles WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.RoleID)
	}
	return out, nil
}

func (r *UserRepository) GetRoleNamesByIDs(ctx context.Context, roleIDs []uuid.UUID) ([]string, error) {
	if len(roleIDs) == 0 {
		return []string{}, nil
	}
	type row struct {
		Name string `gorm:"column:name"`
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Raw(`SELECT name FROM roles WHERE id IN ?`, roleIDs).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Name)
	}
	return out, nil
}

func (r *UserRepository) FindByPatientID(ctx context.Context, patientID uuid.UUID) (*user.User, error) {
	u := &user.User{}
	err := rawGet(ctx, r.db, u,
		`SELECT id, email, company_id, patient_id, password_hash, first_name, last_name, is_active,
                last_login_at, failed_attempts, locked_until, created_at, updated_at, created_by
         FROM users
         WHERE patient_id = $1 AND deleted_at IS NULL
         LIMIT 1`, patientID)
	return u, err
}

func (r *UserRepository) FindSoftDeletedByPatientID(ctx context.Context, patientID uuid.UUID) (*user.User, error) {
	u := &user.User{}
	err := rawGet(ctx, r.db, u,
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

func (r *UserRepository) Reactivate(ctx context.Context, u *user.User) error {
	return rawExec(ctx, r.db,
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
}

func (r *UserRepository) GetRoleIDByName(ctx context.Context, name string) (uuid.UUID, error) {
	var row struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	err := rawGet(ctx, r.db, &row, `SELECT id FROM roles WHERE name = $1 LIMIT 1`, name)
	return row.ID, err
}

func (r *UserRepository) ListProfessionalsMissingWorker(ctx context.Context) ([]*user.User, error) {
	var items []*user.User
	err := r.db.WithContext(ctx).
		Table("users AS u").
		Distinct().
		Select([]string{
			"u.id",
			"u.email",
			"u.company_id",
			"u.patient_id",
			"u.password_hash",
			"u.first_name",
			"u.last_name",
			"u.is_active",
			"u.last_login_at",
			"u.failed_attempts",
			"u.locked_until",
			"u.created_at",
			"u.updated_at",
			"u.created_by",
			"u.deleted_at",
		}).
		Joins("JOIN user_roles ur ON ur.user_id = u.id").
		Joins("JOIN roles ro ON ro.id = ur.role_id AND ro.name = ?", "professional").
		Joins("LEFT JOIN amaur_workers w ON w.user_id = u.id AND w.deleted_at IS NULL").
		Where("u.deleted_at IS NULL").
		Where("w.id IS NULL").
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}
