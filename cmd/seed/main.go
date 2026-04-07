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
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("unwrap db: %v", err)
	}
	defer sqlDB.Close()

	ctx := context.Background()

	var roleRow struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	if err := db.WithContext(ctx).
		Raw(`SELECT id FROM roles WHERE name = 'super_admin' LIMIT 1`).
		Scan(&roleRow).Error; err != nil {
		log.Fatalf("role super_admin not found - run migrations first: %v", err)
	}
	if roleRow.ID == uuid.Nil {
		log.Fatalf("role super_admin not found - run migrations first")
	}

	var existsRow struct {
		Exists bool `gorm:"column:exists"`
	}
	if err := db.WithContext(ctx).
		Raw(`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1) AS exists`, cfg.SeedAdminEmail).
		Scan(&existsRow).Error; err != nil {
		log.Fatalf("check admin exists: %v", err)
	}

	if existsRow.Exists {
		fmt.Printf("Admin user already exists: %s\n", cfg.SeedAdminEmail)
		return
	}

	hashed, err := password.Hash(cfg.SeedAdminPassword)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	userID := uuid.New()
	if err := db.WithContext(ctx).Exec(`
		INSERT INTO users (id, email, password_hash, first_name, last_name, is_active)
		VALUES ($1, $2, $3, $4, $5, true)`,
		userID,
		cfg.SeedAdminEmail,
		hashed,
		cfg.SeedAdminFirstname,
		cfg.SeedAdminLastname,
	).Error; err != nil {
		log.Fatalf("insert user: %v", err)
	}

	if err := db.WithContext(ctx).Exec(
		`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`,
		userID, roleRow.ID,
	).Error; err != nil {
		log.Fatalf("assign role: %v", err)
	}

	fmt.Printf("Admin user created\n")
	fmt.Printf("  Email:    %s\n", cfg.SeedAdminEmail)
	fmt.Printf("  Password: %s\n", cfg.SeedAdminPassword)
}
