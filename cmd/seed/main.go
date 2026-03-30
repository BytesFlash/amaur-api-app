package main

import (
	"context"
	"fmt"
	"log"

	"amaur/api/internal/config"
	"amaur/api/internal/infrastructure/postgres"
	"amaur/api/pkg/password"

	"github.com/google/uuid"
)

func main() {
	cfg := config.Load()

	db, err := postgres.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// ── Asegurar que existe el rol super_admin ─────────────────────────────
	var roleID uuid.UUID
	err = db.QueryRowContext(ctx,
		`SELECT id FROM roles WHERE name = 'super_admin' LIMIT 1`).Scan(&roleID)
	if err != nil {
		log.Fatalf("role super_admin not found — run migrations first: %v", err)
	}

	// ── Verificar que el admin no exista ya ────────────────────────────────
	var exists bool
	_ = db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`,
		cfg.SeedAdminEmail).Scan(&exists)

	if exists {
		fmt.Printf("✓ Admin user already exists: %s\n", cfg.SeedAdminEmail)
		return
	}

	// ── Crear usuario admin ────────────────────────────────────────────────
	hashed, err := password.Hash(cfg.SeedAdminPassword)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	userID := uuid.New()
	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash, first_name, last_name, is_active)
		VALUES ($1, $2, $3, $4, $5, true)`,
		userID,
		cfg.SeedAdminEmail,
		hashed,
		cfg.SeedAdminFirstname,
		cfg.SeedAdminLastname,
	)
	if err != nil {
		log.Fatalf("insert user: %v", err)
	}

	// ── Asignar rol super_admin ────────────────────────────────────────────
	_, err = db.ExecContext(ctx,
		`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`,
		userID, roleID)
	if err != nil {
		log.Fatalf("assign role: %v", err)
	}

	fmt.Printf("✓ Admin user created\n")
	fmt.Printf("  Email:    %s\n", cfg.SeedAdminEmail)
	fmt.Printf("  Password: %s\n", cfg.SeedAdminPassword)
}
