// One-off dev helper: promote a user to teacher by UUID.
// Usage: TUID=<uuid> go run ./scripts/promote_teacher.go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	uid := os.Getenv("TUID")
	if uid == "" {
		fmt.Fprintln(os.Stderr, "TUID required")
		os.Exit(1)
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://asnakech:asnakech@localhost:5432/asnakech?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	var roleID string
	if err := pool.QueryRow(context.Background(), "SELECT id::text FROM roles WHERE code='teacher'").Scan(&roleID); err != nil {
		panic(err)
	}
	if _, err := pool.Exec(context.Background(), "UPDATE users SET role_id=$1 WHERE id=$2", roleID, uid); err != nil {
		panic(err)
	}
	fmt.Println("promoted", uid)
}
