package treatmentplan

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Estados del plan de tratamiento.
const (
	StatusActive    = "active"
	StatusPaused    = "paused"
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
)

// Tipos de frecuencia entre sesiones.
const (
	FrequencyWeekly      = "weekly"       // cada 7 días
	FrequencyTwiceWeekly = "twice_weekly" // cada 3-4 días (~2 veces por semana)
	FrequencyMonthly     = "monthly"      // cada 30 días
	FrequencyCustom      = "custom"       // usar FrequencyInterval explícito
)

// FrequencyIntervalDays retorna el intervalo en días para un tipo de frecuencia.
func FrequencyIntervalDays(frequencyType string, custom int) int {
	switch frequencyType {
	case FrequencyTwiceWeekly:
		return 4
	case FrequencyMonthly:
		return 30
	case FrequencyCustom:
		if custom > 0 {
			return custom
		}
		return 7
	default: // weekly
		return 7
	}
}

// TreatmentPlan representa un plan de tratamiento compuesto por varias sesiones.
type TreatmentPlan struct {
	ID                uuid.UUID  `json:"id"`
	PatientID         uuid.UUID  `json:"patient_id"`
	ProfessionalID    *uuid.UUID `json:"professional_id,omitempty"`
	ServiceTypeID     uuid.UUID  `json:"service_type_id"`
	Title             string     `json:"title"`
	Objective         *string    `json:"objective,omitempty"`
	TotalSessions     int        `json:"total_sessions"`
	CompletedSessions int        `json:"completed_sessions"`
	FrequencyType     string     `json:"frequency_type"`
	FrequencyInterval int        `json:"frequency_interval"`
	StartDate         time.Time  `json:"start_date"`
	EstimatedEndDate  *time.Time `json:"estimated_end_date,omitempty"`
	Status            string     `json:"status"`
	Notes             *string    `json:"notes,omitempty"`
	CreatedBy         *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`

	// Enriquecimientos (JOINs, no columnas reales)
	PatientName      *string `json:"patient_name,omitempty"`
	ProfessionalName *string `json:"professional_name,omitempty"`
	ServiceTypeName  *string `json:"service_type_name,omitempty"`
}

// RemainingSession retorna cuántas sesiones faltan para completar el plan.
func (p *TreatmentPlan) RemainingSessions() int {
	r := p.TotalSessions - p.CompletedSessions
	if r < 0 {
		return 0
	}
	return r
}

// IsTerminal retorna true si el plan no acepta más sesiones.
func (p *TreatmentPlan) IsTerminal() bool {
	return p.Status == StatusCompleted || p.Status == StatusCancelled
}

// Alert representa un indicador operativo de alerta.
type Alert struct {
	Type            string     `json:"type"`
	TreatmentPlanID uuid.UUID  `json:"treatment_plan_id"`
	PatientID       uuid.UUID  `json:"patient_id"`
	PatientName     *string    `json:"patient_name,omitempty"`
	PlanTitle       string     `json:"plan_title"`
	Message         string     `json:"message"`
	SinceDate       *time.Time `json:"since_date,omitempty"`
	SessionsLeft    *int       `json:"sessions_left,omitempty"`
}

// Filter para listar planes de tratamiento.
type Filter struct {
	PatientID      *uuid.UUID
	ProfessionalID *uuid.UUID
	Status         string
	ServiceTypeID  *uuid.UUID
}

// Repository define las operaciones de persistencia del dominio.
type Repository interface {
	Create(ctx context.Context, plan *TreatmentPlan) error
	FindByID(ctx context.Context, id uuid.UUID) (*TreatmentPlan, error)
	Update(ctx context.Context, plan *TreatmentPlan) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f Filter, limit, offset int) ([]*TreatmentPlan, int64, error)

	// RecalculateCompleted actualiza completed_sessions contando desde appointments.
	RecalculateCompleted(ctx context.Context, planID uuid.UUID) error

	// GetAlerts retorna alertas operativas para un profesional (o todos si nil).
	GetAlerts(ctx context.Context, professionalID *uuid.UUID) ([]*Alert, error)
}
