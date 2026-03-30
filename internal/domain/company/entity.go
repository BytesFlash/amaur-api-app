package company

import (
	"time"

	"github.com/google/uuid"
)

type CompanyStatus string
type SizeCategory string

const (
	StatusActive   CompanyStatus = "active"
	StatusInactive CompanyStatus = "inactive"
	StatusProspect CompanyStatus = "prospect"
	StatusChurned  CompanyStatus = "churned"

	SizeMicro  SizeCategory = "micro"
	SizeSmall  SizeCategory = "pequeña"
	SizeMedium SizeCategory = "mediana"
	SizeLarge  SizeCategory = "grande"
)

type Company struct {
	ID              uuid.UUID     `db:"id" json:"id"`
	RUT             *string       `db:"rut" json:"rut,omitempty"`
	Name            string        `db:"name" json:"name"`
	FantasyName     *string       `db:"fantasy_name" json:"fantasy_name,omitempty"`
	Industry        *string       `db:"industry" json:"industry,omitempty"`
	SizeCategory    *SizeCategory `db:"size_category" json:"size_category,omitempty"`
	ContactName     *string       `db:"contact_name" json:"contact_name,omitempty"`
	ContactEmail    *string       `db:"contact_email" json:"contact_email,omitempty"`
	ContactPhone    *string       `db:"contact_phone" json:"contact_phone,omitempty"`
	BillingEmail    *string       `db:"billing_email" json:"billing_email,omitempty"`
	Address         *string       `db:"address" json:"address,omitempty"`
	City            *string       `db:"city" json:"city,omitempty"`
	Region          *string       `db:"region" json:"region,omitempty"`
	Website         *string       `db:"website" json:"website,omitempty"`
	Status          CompanyStatus `db:"status" json:"status"`
	CommercialNotes *string       `db:"commercial_notes" json:"commercial_notes,omitempty"`
	LeadSource      *string       `db:"lead_source" json:"lead_source,omitempty"`
	CreatedAt       time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt       *time.Time    `db:"updated_at" json:"updated_at,omitempty"`
	CreatedBy       *uuid.UUID    `db:"created_by" json:"created_by,omitempty"`
	UpdatedBy       *uuid.UUID    `db:"updated_by" json:"updated_by,omitempty"`
	DeletedAt       *time.Time    `db:"deleted_at" json:"-"`
}

type Branch struct {
	ID           uuid.UUID `db:"id" json:"id"`
	CompanyID    uuid.UUID `db:"company_id" json:"company_id"`
	Name         string    `db:"name" json:"name"`
	Address      *string   `db:"address" json:"address,omitempty"`
	City         *string   `db:"city" json:"city,omitempty"`
	Region       *string   `db:"region" json:"region,omitempty"`
	ContactName  *string   `db:"contact_name" json:"contact_name,omitempty"`
	ContactPhone *string   `db:"contact_phone" json:"contact_phone,omitempty"`
	IsMain       bool      `db:"is_main" json:"is_main"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// PatientSummary is a lightweight view of a patient linked to a company.
type PatientSummary struct {
	ID          uuid.UUID `db:"id"         json:"id"`
	FirstName   string    `db:"first_name" json:"first_name"`
	LastName    string    `db:"last_name"  json:"last_name"`
	RUT         *string   `db:"rut"        json:"rut,omitempty"`
	Email       *string   `db:"email"      json:"email,omitempty"`
	Phone       *string   `db:"phone"      json:"phone,omitempty"`
	Status      string    `db:"status"     json:"status"`
	PatientType string    `db:"patient_type" json:"patient_type"`
	Position    *string   `db:"position"   json:"position,omitempty"`
	Department  *string   `db:"department" json:"department,omitempty"`
}
