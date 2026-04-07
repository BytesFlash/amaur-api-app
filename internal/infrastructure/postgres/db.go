package postgres

import (
	"context"
	"database/sql"
	"strings"
	"time"

	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(dsn string) (*gorm.DB, error) {
	// Append timezone parameter so TIMESTAMPTZ values are returned in Chile local time.
	// This keeps displayed hours aligned with what the user entered regardless of server TZ.
	if !strings.Contains(dsn, "TimeZone=") && !strings.Contains(dsn, "timezone=") {
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn += sep + "TimeZone=America%2FSantiago"
	}

	db, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func sqlDB(db *gorm.DB) (*sql.DB, error) {
	return db.DB()
}
