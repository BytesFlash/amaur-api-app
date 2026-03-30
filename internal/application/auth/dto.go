package auth

import (
	"time"

	"github.com/google/uuid"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         UserDTO   `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type UserDTO struct {
	ID          uuid.UUID  `json:"id"`
	Email       string     `json:"email"`
	CompanyID   *uuid.UUID `json:"company_id,omitempty"`
	PatientID   *uuid.UUID `json:"patient_id,omitempty"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	Roles       []string   `json:"roles"`
	Permissions []string   `json:"permissions"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}
