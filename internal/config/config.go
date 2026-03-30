package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port     string
	Env      string
	LogLevel string

	DatabaseURL string

	JWTSecret        string
	JWTAccessExpiry  time.Duration
	JWTRefreshExpiry time.Duration

	CORSAllowedOrigins string

	StorageProvider  string
	StorageLocalPath string

	SeedAdminEmail     string
	SeedAdminPassword  string
	SeedAdminFirstname string
	SeedAdminLastname  string
}

func Load() *Config {
	_ = godotenv.Load()

	accessExpiry, err := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"))
	if err != nil {
		accessExpiry = 15 * time.Minute
	}
	refreshExpiry, err := time.ParseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"))
	if err != nil {
		refreshExpiry = 7 * 24 * time.Hour
	}

	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnvRequired("DB_NAME")
	dbUser := getEnvRequired("DB_USER")
	dbPass := getEnvRequired("DB_PASS")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")
	databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbUser, dbPass, dbHost, dbPort, dbName, dbSSLMode)

	return &Config{
		Port:     getEnv("BACKEND_PORT", "8080"),
		Env:      getEnv("ENV", "development"),
		LogLevel: getEnv("LOG_LEVEL", "info"),

		DatabaseURL: databaseURL,

		JWTSecret:        getEnvRequired("JWT_SECRET"),
		JWTAccessExpiry:  accessExpiry,
		JWTRefreshExpiry: refreshExpiry,

		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"),

		StorageProvider:  getEnv("STORAGE_PROVIDER", "local"),
		StorageLocalPath: getEnv("STORAGE_LOCAL_PATH", "./uploads"),

		SeedAdminEmail:     getEnv("SEED_ADMIN_EMAIL", "admin@amaur.cl"),
		SeedAdminPassword:  getEnv("SEED_ADMIN_PASSWORD", "ChangeThisNow!2026"),
		SeedAdminFirstname: getEnv("SEED_ADMIN_FIRSTNAME", "Super"),
		SeedAdminLastname:  getEnv("SEED_ADMIN_LASTNAME", "Admin"),
	}
}

func (c *Config) IsProduction() bool { return c.Env == "production" }

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvRequired(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required environment variable not set: " + key)
	}
	return v
}
