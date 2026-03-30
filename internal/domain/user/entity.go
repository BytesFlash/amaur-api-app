package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	Email          string     `db:"email" json:"email"`
	CompanyID      *uuid.UUID `db:"company_id" json:"company_id,omitempty"`
	PatientID      *uuid.UUID `db:"patient_id" json:"patient_id,omitempty"`
	PasswordHash   string     `db:"password_hash" json:"-"`
	FirstName      string     `db:"first_name" json:"first_name"`
	LastName       string     `db:"last_name" json:"last_name"`
	IsActive       bool       `db:"is_active" json:"is_active"`
	LastLoginAt    *time.Time `db:"last_login_at" json:"last_login_at,omitempty"`
	FailedAttempts int        `db:"failed_attempts" json:"-"`
	LockedUntil    *time.Time `db:"locked_until" json:"-"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `db:"updated_at" json:"updated_at,omitempty"`
	CreatedBy      *uuid.UUID `db:"created_by" json:"created_by,omitempty"`
	DeletedAt      *time.Time `db:"deleted_at" json:"-"`
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

func (u *User) IsLocked() bool {
	return u.LockedUntil != nil && u.LockedUntil.After(time.Now())
}
