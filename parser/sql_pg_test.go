package parser

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func startPostgres(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker is required for PostgreSQL integration tests")
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("docker daemon is not running")
	}

	name := fmt.Sprintf("berm-sql-test-%d", time.Now().UnixNano())
	run := exec.Command(
		"docker", "run", "-d", "--rm",
		"--name", name,
		"-e", "POSTGRES_USER=test",
		"-e", "POSTGRES_PASSWORD=test",
		"-e", "POSTGRES_DB=berm",
		"postgres:16-alpine",
	)
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("docker run: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", name).Run()
	})

	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		if exec.Command("docker", "exec", name, "pg_isready", "-U", "test", "-d", "berm").Run() == nil {
			return name
		}
		time.Sleep(400 * time.Millisecond)
	}
	logs, _ := exec.Command("docker", "logs", name).CombinedOutput()
	t.Fatalf("postgres did not become ready\n%s", logs)
	return ""
}

func psql(t *testing.T, container, query string) string {
	t.Helper()
	cmd := exec.Command(
		"docker", "exec", "-i", container,
		"psql", "-U", "test", "-d", "berm", "-v", "ON_ERROR_STOP=1", "-At", "-c", query,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("psql %q: %v\n%s", query, err, out)
	}
	return string(out)
}

func TestPostgresCreateAndQuery(t *testing.T) {
	container := startPostgres(t)
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

	ddl := p.GenerateCreationSQL("user")
	if ddl == "" {
		t.Fatal("expected CREATE TABLE SQL")
	}
	psql(t, container, ddl)

	got := strings.TrimSpace(psql(t, container, `
		SELECT column_name || ' ' || udt_name
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'user'
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

	psql(t, container, `
		INSERT INTO "user" (
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

	row := strings.TrimSpace(psql(t, container, `SELECT name || ' ' || active::text || ' ' || score::text FROM "user" WHERE id = '550e8400-e29b-41d4-a716-446655440000'`))
	if row != "Ada true 1.5" {
		t.Fatalf("selected row = %q, want %q", row, "Ada true 1.5")
	}
}
