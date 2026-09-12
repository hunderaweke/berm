package parser

import "testing"

func TestIsTargetEmbeddedStruct(t *testing.T) {
	p := parseModule(t, map[string]string{
		"user.go": `package parsertest

import "github.com/hunderaweke/berm/models"

type TestEmbeddedStruct struct {
	models.Model
	ID        string
	Name      string
	Number    int
	CreatedAt int
}
`,
	})

	info := mustStruct(t, p, "test_embedded_struct")
	if info.Name != "TestEmbeddedStruct" {
		t.Fatalf("Name = %q", info.Name)
	}
	requireFields(t, info.Fields, "ID", "CreatedAt", "UpdatedAt", "ID", "Name", "Number", "CreatedAt")
}
