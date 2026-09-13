package parser

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func startPostgres(t *testing.T) *pgx.Conn {
	t.Helper()
	ctx := context.Background()
	pgContainer, err := postgres.Run(
		ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("berm"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := pgContainer.Terminate(context.Background()); err != nil {
			t.Errorf("failed to terminate postgres container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get postgres connection string: %v", err)
	}
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Close(context.Background()); err != nil {
			t.Errorf("failed to close postgres connection: %v", err)
		}
	})
	return conn
}

func psql(t *testing.T, conn *pgx.Conn, query string) string {
	t.Helper()
	rows, err := conn.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("failed to query: %v\n%s", err, query)
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			t.Fatalf("failed to read row: %v", err)
		}
		parts := make([]string, len(values))
		for i, v := range values {
			parts[i] = fmt.Sprint(v)
		}
		lines = append(lines, strings.Join(parts, ""))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to iterate rows: %v", err)
	}
	return strings.Join(lines, "\n")
}

func TestPostgresCreateAndQuery(t *testing.T) {
	conn := startPostgres(t)
	p := parseModule(t, map[string]string{
		"schema.go": `package parsertest

import (
	"time"

	"github.com/google/uuid"
	"github.com/hunderaweke/berm/models"
)

type User struct {
	models.Model
	Name        string
	Active      bool
	Score       float64
	Tags        []string
	Metadata    map[string]any
	Avatar      []byte
	UserID      uuid.UUID
	PublishedAt *time.Time
	CreatedAt   time.Time
}
`,
	})

	ddl := p.GenerateCreationSQL("users")
	if ddl == "" {
		t.Fatal("expected CREATE TABLE SQL")
	}
	psql(t, conn, ddl)

	got := strings.TrimSpace(psql(t, conn, `
		SELECT column_name || ' ' || udt_name
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'users'
		ORDER BY ordinal_position
	`))
	want := []string{
		"id uuid",
		"created_at timestamp",
		"updated_at timestamp",
		"name text",
		"active bool",
		"score float4",
		"tags _text",
		"metadata jsonb",
		"avatar bytea",
		"user_id uuid",
		"published_at timestamp",
	}
	lines := strings.Split(got, "\n")
	if len(lines) != len(want) {
		t.Fatalf("columns = %q, want %v", got, want)
	}
	for i, col := range want {
		if lines[i] != col {
			t.Fatalf("column[%d] = %q, want %q\nall:\n%s", i, lines[i], col, got)
		}
	}

	psql(t, conn, `
		INSERT INTO "users" (
			id, created_at, updated_at, name, active, score, tags, metadata, avatar, user_id, published_at
		) VALUES (
			'550e8400-e29b-41d4-a716-446655440000',
			'2026-09-12 15:00:00',
			'2026-09-12 16:00:00',
			'Ada',
			true,
			1.5,
			'{go,sql}',
			'{"ok":true}',
			'\xDEAD',
			'11111111-1111-1111-1111-111111111111',
			'2026-09-13 09:00:00'
		)
	`)

	row := strings.TrimSpace(psql(t, conn, `SELECT name || ' ' || active::text || ' ' || score::text FROM "users" WHERE id = '550e8400-e29b-41d4-a716-446655440000'`))
	if row != "Ada true 1.5" {
		t.Fatalf("selected row = %q, want %q", row, "Ada true 1.5")
	}
}

func TestPostgresInsert(t *testing.T) {
	conn := startPostgres(t)
	p := parseModule(t, map[string]string{
		"schema.go": `package parsertest

import (
	"github.com/hunderaweke/berm/models"
)

type User struct {
	models.Model
	Name string
	Age  int
}
`,
	})

	ddl := p.GenerateCreationSQL("users")
	if ddl == "" {
		t.Fatal("expected CREATE TABLE SQL")
	}
	psql(t, conn, ddl)
	sql := p.GenerateInsertSQL("users", map[string]any{
		"name": "John",
		"age":  30,
	})
	if sql == "" {
		t.Fatal("expected INSERT SQL")
	}
	psql(t, conn, sql)
	row := strings.TrimSpace(psql(t, conn, `SELECT name || ' ' || age::text FROM "users" WHERE name = 'John'`))
	if row != "John 30" {
		t.Fatalf("selected row = %q, want %q", row, "John 30")
	}
}

func TestPostgresBatchInsert(t *testing.T) {
	conn := startPostgres(t)
	p := parseModule(t, map[string]string{
		"schema.go": `package parsertest

import "github.com/hunderaweke/berm/models"

type User struct {
	models.Model
	Name string
	Age  int
}
`,
	})
	psql(t, conn, p.GenerateCreationSQL("users"))
	sql := p.GenerateBatchInsertSQL("users", []map[string]any{
		{"name": "John", "age": 30},
		{"name": "Jane", "age": 25},
		{"name": "Ada"},
	})
	if sql == "" {
		t.Fatal("expected batch INSERT SQL")
	}
	if strings.Count(sql, "INSERT INTO") != 1 {
		t.Fatalf("expected one statement:\n%s", sql)
	}
	psql(t, conn, sql)

	count := strings.TrimSpace(psql(t, conn, `SELECT count(*)::text FROM "users"`))
	if count != "3" {
		t.Fatalf("row count = %q, want 3\n%s", count, sql)
	}
	got := strings.TrimSpace(psql(t, conn, `SELECT name || ' ' || coalesce(age::text, 'null') FROM "users" ORDER BY name`))
	if got != "Ada null\nJane 25\nJohn 30" {
		t.Fatalf("rows = %q\n%s", got, sql)
	}
}
