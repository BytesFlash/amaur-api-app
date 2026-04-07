package postgres

import (
	"context"
	"database/sql"
	"errors"

	"gorm.io/gorm"
)

func rawExec(ctx context.Context, db *gorm.DB, query string, args ...interface{}) error {
	return db.WithContext(ctx).Exec(query, args...).Error
}

func rawGet[T any](ctx context.Context, db *gorm.DB, dest *T, query string, args ...interface{}) error {
	tx := db.WithContext(ctx).Raw(query, args...).Scan(dest)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func rawSelect[T any](ctx context.Context, db *gorm.DB, dest *[]T, query string, args ...interface{}) error {
	return db.WithContext(ctx).Raw(query, args...).Scan(dest).Error
}

func rawSelectPtr[T any](ctx context.Context, db *gorm.DB, dest *[]*T, query string, args ...interface{}) error {
	return db.WithContext(ctx).Raw(query, args...).Scan(dest).Error
}

func withTx(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

func isNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, gorm.ErrRecordNotFound)
}
