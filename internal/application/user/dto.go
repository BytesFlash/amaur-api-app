package user

import (
	"time"

	"github.com/google/uuid"
)

// CreateUserRequest is used by admins to create a new user.
type CreateUserRequest struct {
	Email             string      `json:"email"     validate:"required,email"`
	Password          string      `json:"password"  validate:"required,min=8"`
	FirstName         string      `json:"first_name" validate:"required"`
	LastName          string      `json:"last_name"  validate:"required"`
	RUT               *string     `json:"rut"`
	Phone             *string     `json:"phone"`
	RoleTitle         *string     `json:"role_title"`
	Specialty         *string     `json:"specialty"`
	AvailabilityNotes *string     `json:"availability_notes"`
	CompanyID         *uuid.UUID  `json:"company_id"`
	PatientID         *uuid.UUID  `json:"patient_id"`
	RoleIDs           []uuid.UUID `json:"role_ids"`
}

// UpdateUserRequest for partial edits.
type UpdateUserRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	IsActive  *bool   `json:"is_active"`
}

// ChangePasswordRequest for the admin to reset a user's password.
type ChangePasswordRequest struct {
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// AssignRolesRequest replaces the user's roles.
type AssignRolesRequest struct {
	RoleIDs []uuid.UUID `json:"role_ids" validate:"required"`
}

// UserDTO is the public representation of a user.
type UserDTO struct {
	ID          uuid.UUID  `json:"id"`
	Email       string     `json:"email"`
	CompanyID   *uuid.UUID `json:"company_id,omitempty"`
	PatientID   *uuid.UUID `json:"patient_id,omitempty"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	IsActive    bool       `json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	Roles       []string   `json:"roles"`
	CreatedAt   time.Time  `json:"created_at"`
}

// RoleDTO is the public representation of a role.
type RoleDTO struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"is_system"`
}
