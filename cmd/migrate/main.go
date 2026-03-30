package main

import (
"database/sql"
"fmt"
"os"

_ "github.com/lib/pq"
)

func main() {
dsn := os.Getenv("DATABASE_URL")
if dsn == "" {
dsn = "postgres://postgres:123456@localhost:5432/amaur-api?sslmode=disable"
}

db, err := sql.Open("postgres", dsn)
if err != nil {
fmt.Fprintf(os.Stderr, "open: %v\n", err)
os.Exit(1)
}
defer db.Close()

sqls := []string{
`ALTER TABLE company_program_schedule_rules ADD COLUMN IF NOT EXISTS worker_id UUID REFERENCES amaur_workers(id) ON DELETE SET NULL`,
`CREATE INDEX IF NOT EXISTS idx_program_rules_worker ON company_program_schedule_rules(worker_id)`,
`INSERT INTO schema_migrations (version, dirty) VALUES (24, false) ON CONFLICT (version) DO NOTHING`,
}

for _, s := range sqls {
if _, err := db.Exec(s); err != nil {
fmt.Fprintf(os.Stderr, "WARN: %v\n", err)
} else {
fmt.Println("OK:", s[:min(70, len(s))])
}
}
fmt.Println("Done.")
}

func min(a, b int) int {
if a < b {
return a
}
return b
}
