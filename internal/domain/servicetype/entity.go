package servicetype

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SpecialtyItem is a specialty catalog entry associated to this service type.
type SpecialtyItem struct {
	Code string `db:"code" json:"code"`
	Name string `db:"name" json:"name"`
}

// ServiceType represents a type of clinical or wellness service AMAUR provides.
type ServiceType struct {
	ID                     uuid.UUID  `db:"id"                       json:"id"`
	Name                   string     `db:"name"                     json:"name"`
	Category               *string    `db:"category"                 json:"category,omitempty"`
	Description            *string    `db:"description"              json:"description,omitempty"`
	DefaultDurationMinutes *int       `db:"default_duration_minutes" json:"default_duration_minutes,omitempty"`
	IsGroupService         bool       `db:"is_group_service"         json:"is_group_service"`
	RequiresClinicalRecord bool       `db:"requires_clinical_record" json:"requires_clinical_record"`
	IsActive               bool       `db:"is_active"                json:"is_active"`
	CreatedAt              time.Time  `db:"created_at"               json:"created_at"`
	UpdatedAt              *time.Time `db:"updated_at"               json:"updated_at,omitempty"`

	// Populated from service_type_specialties join (not a DB column).
	Specialties []SpecialtyItem `db:"-" gorm:"-" json:"specialties,omitempty"`
}

// Repository defines the persistence contract for service types.
type Repository interface {
	List(ctx context.Context, activeOnly bool) ([]*ServiceType, error)
	FindByID(ctx context.Context, id uuid.UUID) (*ServiceType, error)
	Create(ctx context.Context, st *ServiceType) error
	Update(ctx context.Context, st *ServiceType) error
	// SetSpecialties atomically replaces the specialties linked to a service type.
	SetSpecialties(ctx context.Context, serviceTypeID uuid.UUID, codes []string) error
}
