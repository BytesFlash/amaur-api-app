package jwtpkg

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID      uuid.UUID  `json:"sub"`
	Email       string     `json:"email"`
	CompanyID   *uuid.UUID `json:"company_id,omitempty"`
	PatientID   *uuid.UUID `json:"patient_id,omitempty"`
	Roles       []string   `json:"roles"`
	Permissions []string   `json:"permissions"`
	jwt.RegisteredClaims
}

func (c *Claims) HasPermission(perm string) bool {
	for _, p := range c.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

func (c *Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

type Manager struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewManager(secret string, accessExpiry, refreshExpiry time.Duration) *Manager {
	return &Manager{
		secret:        []byte(secret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (m *Manager) GenerateAccessToken(userID uuid.UUID, email string, companyID *uuid.UUID, patientID *uuid.UUID, roles, permissions []string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:      userID,
		Email:       email,
		CompanyID:   companyID,
		PatientID:   patientID,
		Roles:       roles,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessExpiry)),
			ID:        uuid.New().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *Manager) GenerateRefreshToken() string {
	return uuid.New().String()
}

func (m *Manager) RefreshExpiry() time.Duration {
	return m.refreshExpiry
}

func (m *Manager) ParseAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
