package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"github.com/arrowwhi/books_review_bot/migrations"
)

func main() {
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx) //nolint:errcheck

	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT        PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("create schema_migrations: %v", err)
	}

	rows, err := conn.Query(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		log.Fatalf("query applied: %v", err)
	}
	applied := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			log.Fatalf("scan: %v", err)
		}
		applied[v] = true
	}
	rows.Close()

	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		log.Fatalf("read migrations: %v", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	applied_count := 0
	for _, name := range files {
		if applied[name] {
			fmt.Printf("skip  %s\n", name)
			continue
		}

		data, err := migrations.FS.ReadFile(name)
		if err != nil {
			log.Fatalf("read %s: %v", name, err)
		}

		sql := extractUp(string(data))
		if sql == "" {
			fmt.Printf("skip  %s (no Up section)\n", name)
			continue
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			log.Fatalf("begin tx for %s: %v", name, err)
		}

		if _, err := tx.Exec(ctx, sql); err != nil {
			_ = tx.Rollback(ctx)
			log.Fatalf("apply %s: %v", name, err)
		}

		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			log.Fatalf("record %s: %v", name, err)
		}

		if err := tx.Commit(ctx); err != nil {
			log.Fatalf("commit %s: %v", name, err)
		}

		fmt.Printf("apply %s OK\n", name)
		applied_count++
	}

	if applied_count == 0 {
		fmt.Println("nothing to apply")
	} else {
		fmt.Printf("done (%d applied)\n", applied_count)
	}
}

// extractUp извлекает SQL между маркерами "-- +goose Up" и "-- +goose Down".
// Если маркеров нет — возвращает весь файл как есть.
func extractUp(content string) string {
	const upMarker = "-- +goose Up"
	const downMarker = "-- +goose Down"

	start := strings.Index(content, upMarker)
	if start == -1 {
		return strings.TrimSpace(content)
	}
	start += len(upMarker)

	rest := content[start:]
	end := strings.Index(rest, downMarker)
	if end == -1 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:end])
}
