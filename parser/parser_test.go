package parser

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func bermDir(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/hunderaweke/berm").CombinedOutput()
	if err != nil {
		t.Fatalf("resolve berm module dir: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeTestModule(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	goMod := "module parsertest\n\n" +
		"go 1.22\n\n" +
		"require github.com/hunderaweke/berm v0.0.0\n\n" +
		"replace github.com/hunderaweke/berm => " + bermDir(t) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for name, src := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, out)
	}
	return dir
}

func parseModule(t *testing.T, files map[string]string) *Parser {
	t.Helper()
	p := NewParser(writeTestModule(t, files))
	if err := p.Parse(); err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}
	return p
}

func mustStruct(t *testing.T, p *Parser, tableName string) StructInfo {
	t.Helper()
	info, ok := p.Structs[tableName]
	if !ok {
		keys := make([]string, 0, len(p.Structs))
		for k := range p.Structs {
			keys = append(keys, k)
		}
		t.Fatalf("missing struct table %q; have %v", tableName, keys)
	}
	return info
}

func fieldNames(fields []FieldInfo) []string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}
	return names
}

func mustField(t *testing.T, fields []FieldInfo, name string) FieldInfo {
	t.Helper()
	for _, f := range fields {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("missing field %q; have %v", name, fieldNames(fields))
	return FieldInfo{}
}

func requireFields(t *testing.T, fields []FieldInfo, want ...string) {
	t.Helper()
	got := fieldNames(fields)
	if !slices.Equal(got, want) {
		t.Fatalf("fields = %v, want %v", got, want)
	}
}

func TestNewParser(t *testing.T) {
	p := NewParser("/tmp/example")
	if p.Dir != "/tmp/example" {
		t.Fatalf("Dir = %q", p.Dir)
	}
	if p.Structs == nil {
		t.Fatal("Structs map is nil")
	}
	if len(p.Structs) != 0 {
		t.Fatalf("Structs should start empty, got %d", len(p.Structs))
	}
}

func TestParse_BasicModelEmbed(t *testing.T) {
	p := parseModule(t, map[string]string{
		"user.go": `package parsertest

import "github.com/hunderaweke/berm/models"

type User struct {
	models.Model
	Name string
}
`,
	})

	info := mustStruct(t, p, "user")
	if info.Name != "User" || info.TableName != "user" {
		t.Fatalf("struct identity = %+v", info)
	}
	requireFields(t, info.Fields, "ID", "CreatedAt", "UpdatedAt", "Name")

	id := mustField(t, info.Fields, "ID")
	if id.Parent != "Model" {
		t.Fatalf("ID.Parent = %q, want Model", id.Parent)
	}
	if !strings.Contains(id.Tag, `json:"id"`) {
		t.Fatalf("ID.Tag = %q", id.Tag)
	}
	if ResolveSQLType(id) != SQLTypeUUID {
		t.Fatalf("ID sql type = %s", ResolveSQLType(id))
	}
	if ResolveSQLType(mustField(t, info.Fields, "CreatedAt")) != SQLTypeTimestamp {
		t.Fatal("CreatedAt should be TIMESTAMP")
	}
	name := mustField(t, info.Fields, "Name")
	if name.Parent != "" || name.Embedded {
		t.Fatalf("Name should be a top-level field, got %+v", name)
	}
	if name.Type != "string" {
		t.Fatalf("Name.Type = %q", name.Type)
	}
}

func TestParse_PointerModelEmbed(t *testing.T) {
	p := parseModule(t, map[string]string{
		"user.go": `package parsertest

import "github.com/hunderaweke/berm/models"

type Account struct {
	*models.Model
	Email string
}
`,
	})

	info := mustStruct(t, p, "account")
	requireFields(t, info.Fields, "ID", "CreatedAt", "UpdatedAt", "Email")
	if mustField(t, info.Fields, "ID").Parent != "Model" {
		t.Fatal("pointer embed should still flatten Model fields")
	}
}

func TestParse_ModelNotFirstField(t *testing.T) {
	p := parseModule(t, map[string]string{
		"user.go": `package parsertest

import "github.com/hunderaweke/berm/models"

type User struct {
	Name string
	models.Model
	Age int
}
`,
	})

	info := mustStruct(t, p, "user")
	requireFields(t, info.Fields, "Name", "ID", "CreatedAt", "UpdatedAt", "Age")
}

func TestParse_NamedModelFieldIgnored(t *testing.T) {
	p := parseModule(t, map[string]string{
		"user.go": `package parsertest

import "github.com/hunderaweke/berm/models"

type User struct {
	Base models.Model
	Name string
}
`,
	})
	if len(p.Structs) != 0 {
		t.Fatalf("named Model field must not be treated as an embed: %+v", p.Structs)
	}
}

func TestParse_NoModelEmbedIgnored(t *testing.T) {
	p := parseModule(t, map[string]string{
		"plain.go": `package parsertest

type Address struct {
	Street string
	City   string
}

type NotAStruct int
`,
	})
	if len(p.Structs) != 0 {
		t.Fatalf("expected no collected structs, got %+v", p.Structs)
	}
}

func TestParse_DefinedTypeWrappingModelIgnored(t *testing.T) {
	p := parseModule(t, map[string]string{
		"user.go": `package parsertest

import "github.com/hunderaweke/berm/models"

type Base models.Model

type User struct {
	Base
	Name string
}
`,
	})
	if len(p.Structs) != 0 {
		t.Fatalf("defined type wrapping Model is not models.Model: %+v", p.Structs)
	}
}

func TestParse_TypeAliasToModelNotCollected(t *testing.T) {
	p := parseModule(t, map[string]string{
		"user.go": `package parsertest

import "github.com/hunderaweke/berm/models"

type Base = models.Model

type AliasUser struct {
	Base
	Name string
}
`,
	})
	if len(p.Structs) != 0 {
		t.Fatalf("alias embeds are *types.Alias, which the parser does not unwrap: %+v", p.Structs)
	}
}

func TestParse_IndirectModelEmbedNotCollected(t *testing.T) {
	p := parseModule(t, map[string]string{
		"user.go": `package parsertest

import "github.com/hunderaweke/berm/models"

type Profile struct {
	models.Model
	Bio string
}

type User struct {
	Profile
	Name string
}
`,
	})

	if _, ok := p.Structs["user"]; ok {
		t.Fatal("User only embeds Profile, not models.Model, and should not be collected")
	}
	info := mustStruct(t, p, "profile")
	requireFields(t, info.Fields, "ID", "CreatedAt", "UpdatedAt", "Bio")
}

func TestParse_LocalNestedAndPointerEmbeds(t *testing.T) {
	p := parseModule(t, map[string]string{
		"user.go": `package parsertest

import (
	"time"

	"github.com/hunderaweke/berm/models"
)

type SoftDelete struct {
	DeletedAt *time.Time ` + "`json:\"deleted_at\"`" + `
}

type Audit struct {
	SoftDelete
	Version int ` + "`json:\"version\"`" + `
}

type Company struct {
	models.Model
	*Audit
	Title string
}
`,
	})

	info := mustStruct(t, p, "company")
	requireFields(t, info.Fields, "ID", "CreatedAt", "UpdatedAt", "DeletedAt", "Version", "Title")

	deleted := mustField(t, info.Fields, "DeletedAt")
	if deleted.Parent != "SoftDelete" {
		t.Fatalf("DeletedAt.Parent = %q, want SoftDelete", deleted.Parent)
	}
	if !strings.Contains(deleted.Tag, `json:"deleted_at"`) {
		t.Fatalf("DeletedAt.Tag = %q", deleted.Tag)
	}
	if ResolveSQLType(deleted) != SQLTypeTimestamp {
		t.Fatalf("DeletedAt sql type = %s", ResolveSQLType(deleted))
	}

	version := mustField(t, info.Fields, "Version")
	if version.Parent != "Audit" {
		t.Fatalf("Version.Parent = %q, want Audit", version.Parent)
	}
	if ResolveSQLType(version) != SQLTypeInteger {
		t.Fatalf("Version sql type = %s", ResolveSQLType(version))
	}
}

func TestParse_KitchenSinkFieldTypes(t *testing.T) {
	p := parseModule(t, map[string]string{
		"kitchen.go": `package parsertest

import (
	"time"

	"github.com/google/uuid"
	"github.com/hunderaweke/berm/models"
)

type Address struct {
	Street string
	Geo    map[string]float64
}

type Kitchen struct {
	models.Model
	Title       string
	Active      bool
	Score       float64
	Count       uint32
	Tags        []string
	Metadata    map[string]any
	Settings    map[string]string
	Times       map[string]time.Time
	Avatar      []byte
	UserID      uuid.UUID
	PublishedAt *time.Time
	Address     Address
	Home        *Address
	Addresses   []Address
	Notes       *[]string
	Raw         **int
	secret      string
	Name, Email string
	Inline      struct{ X int }
}
`,
	})

	info := mustStruct(t, p, "kitchen")
	requireFields(t, info.Fields,
		"ID", "CreatedAt", "UpdatedAt",
		"Title", "Active", "Score", "Count",
		"Tags", "Metadata", "Settings", "Times",
		"Avatar", "UserID", "PublishedAt",
		"Address", "Home", "Addresses", "Notes", "Raw",
		"secret", "Name", "Email", "Inline",
	)

	cases := []struct {
		name    string
		sql     SQLType
		typeHas string
		parent  string
	}{
		{name: "Title", sql: SQLTypeText, typeHas: "string"},
		{name: "Active", sql: SQLTypeBoolean, typeHas: "bool"},
		{name: "Score", sql: SQLTypeReal, typeHas: "float64"},
		{name: "Count", sql: SQLTypeInteger, typeHas: "uint32"},
		{name: "Tags", sql: SQLTypeArray, typeHas: "[]string"},
		{name: "Metadata", sql: SQLTypeJSONB, typeHas: "map[string]"},
		{name: "Settings", sql: SQLTypeJSONB, typeHas: "map[string]string"},
		{name: "Times", sql: SQLTypeJSONB, typeHas: "map[string]time.Time"},
		{name: "Avatar", sql: SQLTypeByte, typeHas: "[]byte"},
		{name: "UserID", sql: SQLTypeUUID, typeHas: "uuid.UUID"},
		{name: "PublishedAt", sql: SQLTypeTimestamp, typeHas: "*time.Time"},
		{name: "Address", sql: SQLTypeText, typeHas: "Address"},
		{name: "Home", sql: SQLTypeText, typeHas: "*parsertest.Address"},
		{name: "Addresses", sql: SQLTypeArray, typeHas: "[]parsertest.Address"},
		{name: "Notes", sql: SQLTypeArray, typeHas: "*[]string"},
		{name: "Raw", sql: SQLTypeInteger, typeHas: "**int"},
		{name: "secret", sql: SQLTypeText, typeHas: "string"},
		{name: "Name", sql: SQLTypeText, typeHas: "string"},
		{name: "Email", sql: SQLTypeText, typeHas: "string"},
		{name: "Inline", sql: SQLTypeText, typeHas: "struct"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := mustField(t, info.Fields, tc.name)
			if got := ResolveSQLType(f); got != tc.sql {
				t.Fatalf("ResolveSQLType(%s) = %s, want %s (type %q)", tc.name, got, tc.sql, f.Type)
			}
			if !strings.Contains(f.Type, tc.typeHas) {
				t.Fatalf("%s.Type = %q, want substring %q", tc.name, f.Type, tc.typeHas)
			}
			if f.Parent != tc.parent {
				t.Fatalf("%s.Parent = %q, want %q", tc.name, f.Parent, tc.parent)
			}
			if f.Embedded {
				t.Fatalf("%s should not be marked embedded", tc.name)
			}
		})
	}
}

func TestParse_MultipleStructsAndFiles(t *testing.T) {
	p := parseModule(t, map[string]string{
		"first.go": `package parsertest

import "github.com/hunderaweke/berm/models"

type FirstStruct struct {
	models.Model
	Name string
}
`,
		"second.go": `package parsertest

import "github.com/hunderaweke/berm/models"

type HTTPClient struct {
	models.Model
	Endpoint string
}

type ignored struct {
	Value string
}
`,
		"README.md": "# not go",
	})

	if len(p.Structs) != 2 {
		t.Fatalf("got %d structs, want 2: %+v", len(p.Structs), p.Structs)
	}
	if mustStruct(t, p, "first_struct").Name != "FirstStruct" {
		t.Fatal("first_struct name")
	}
	httpClient := mustStruct(t, p, "http_client")
	if httpClient.Name != "HTTPClient" {
		t.Fatalf("HTTPClient table mapping = %+v", httpClient)
	}
	if _, ok := p.Structs["ignored"]; ok {
		t.Fatal("struct without Model embed should not be collected")
	}
}

func TestParse_SubpackageIsIgnored(t *testing.T) {
	p := parseModule(t, map[string]string{
		"root.go": `package parsertest

import "github.com/hunderaweke/berm/models"

type Root struct {
	models.Model
	Name string
}
`,
		"nested/other.go": `package nested

import "github.com/hunderaweke/berm/models"

type Nested struct {
	models.Model
	Name string
}
`,
	})

	if len(p.Structs) != 1 {
		t.Fatalf("parser should only load the root package, got %+v", p.Structs)
	}
	mustStruct(t, p, "root")
}

func TestParse_UnexportedStructStillCollected(t *testing.T) {
	p := parseModule(t, map[string]string{
		"user.go": `package parsertest

import "github.com/hunderaweke/berm/models"

type privateUser struct {
	models.Model
	name string
}
`,
	})

	info := mustStruct(t, p, "private_user")
	if info.Name != "privateUser" {
		t.Fatalf("Name = %q", info.Name)
	}
	requireFields(t, info.Fields, "ID", "CreatedAt", "UpdatedAt", "name")
}

func TestParse_NonStructDeclarationsIgnored(t *testing.T) {
	p := parseModule(t, map[string]string{
		"misc.go": `package parsertest

import "github.com/hunderaweke/berm/models"

type Thing interface {
	Name() string
}

type Count int

type CountAlias = int

var _ = struct {
	models.Model
}{}
`,
	})
	if len(p.Structs) != 0 {
		t.Fatalf("interfaces, defined types, and anonymous structs should be ignored, got %+v", p.Structs)
	}
}

func TestParse_FunctionLocalStructIsCollected(t *testing.T) {
	p := parseModule(t, map[string]string{
		"misc.go": `package parsertest

import "github.com/hunderaweke/berm/models"

func helper() {
	type local struct {
		models.Model
		Name string
	}
	_ = local{}
}
`,
	})

	info := mustStruct(t, p, "local")
	if info.Name != "local" {
		t.Fatalf("Name = %q", info.Name)
	}
	requireFields(t, info.Fields, "ID", "CreatedAt", "UpdatedAt", "Name")
}

func TestParse_EmptyPackage(t *testing.T) {
	p := parseModule(t, map[string]string{
		"empty.go": "package parsertest\n",
	})
	if len(p.Structs) != 0 {
		t.Fatalf("empty package should yield no structs, got %+v", p.Structs)
	}
}

func TestParse_InvalidSource(t *testing.T) {
	dir := writeTestModule(t, map[string]string{
		"bad.go": `package parsertest

type User struct {
	models.Model
	Name string
}
`,
	})
	p := NewParser(dir)
	if err := p.Parse(); err == nil {
		t.Fatal("Parse() should fail on unresolved types / invalid source")
	}
}

func TestParse_MissingDirectory(t *testing.T) {
	p := NewParser(filepath.Join(t.TempDir(), "does-not-exist"))
	if err := p.Parse(); err == nil {
		t.Fatal("Parse() should fail for a missing directory")
	}
}

func TestParse_DoesNotKeepEmbeddedStructAsColumn(t *testing.T) {
	p := parseModule(t, map[string]string{
		"user.go": `package parsertest

import "github.com/hunderaweke/berm/models"

type User struct {
	models.Model
	Name string
}
`,
	})

	for _, f := range mustStruct(t, p, "user").Fields {
		if f.Name == "Model" {
			t.Fatal("embedded Model should be flattened, not kept as its own column")
		}
		if f.Embedded {
			t.Fatalf("flattened fields should not stay marked embedded: %+v", f)
		}
	}
}

func TestParse_GenericStructWithModel(t *testing.T) {
	p := parseModule(t, map[string]string{
		"box.go": `package parsertest

import "github.com/hunderaweke/berm/models"

type Box[T any] struct {
	models.Model
	Value T
}
`,
	})

	info := mustStruct(t, p, "box")
	requireFields(t, info.Fields, "ID", "CreatedAt", "UpdatedAt", "Value")
	value := mustField(t, info.Fields, "Value")
	if value.Type == "" {
		t.Fatal("generic Value field should still have a type string")
	}
}
