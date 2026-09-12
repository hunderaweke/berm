package parser

import (
	"strings"
	"testing"
)

func TestResolveTableName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{
			name:     "Simple",
			expected: "simple",
		},
		{
			name:     "User",
			expected: "user",
		},
		{
			name:     "VeryLongName",
			expected: "very_long_name",
		},
		{
			name:     "HTTPClient",
			expected: "http_client",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveTableName(FieldInfo{
				Name:     tt.name,
				Type:     "string",
				Tag:      "",
				Embedded: false,
				Parent:   "",
			})
			if got != tt.expected {
				t.Errorf("ResolveTableName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestResolveSQLType(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		expected SQLType
	}{
		{name: "string", typeName: "string", expected: SQLTypeText},
		{name: "ptr_string", typeName: "*string", expected: SQLTypeText},
		{name: "int", typeName: "int", expected: SQLTypeInteger},
		{name: "int64", typeName: "int64", expected: SQLTypeInteger},
		{name: "uint32", typeName: "uint32", expected: SQLTypeInteger},
		{name: "float32", typeName: "float32", expected: SQLTypeReal},
		{name: "float64", typeName: "float64", expected: SQLTypeReal},
		{name: "bool", typeName: "bool", expected: SQLTypeBoolean},
		{name: "byte", typeName: "byte", expected: SQLTypeByte},

		// Slices and arrays
		{name: "string_array", typeName: "[]string", expected: SQLTypeArray},
		{name: "int_array", typeName: "[]int", expected: SQLTypeArray},
		{name: "float_array", typeName: "[]float64", expected: SQLTypeArray},
		{name: "bool_array", typeName: "[]bool", expected: SQLTypeArray},
		{name: "interface_array", typeName: "[]interface{}", expected: SQLTypeArray},
		{name: "any_array", typeName: "[]any", expected: SQLTypeArray},

		// UUIDs
		{name: "uuid_struct", typeName: "uuid.UUID", expected: SQLTypeUUID},
		{name: "uuid_upper", typeName: "UUID", expected: SQLTypeUUID},

		// Times and dates
		{name: "time_struct", typeName: "time.Time", expected: SQLTypeTimestamp},
		{name: "ptr_time", typeName: "*time.Time", expected: SQLTypeTimestamp},
		{name: "timestamp", typeName: "timestamp", expected: SQLTypeTimestamp},
		{name: "datetime", typeName: "datetime", expected: SQLTypeTimestamp},
		{name: "Time", typeName: "Time", expected: SQLTypeTimestamp},
		{name: "date", typeName: "date", expected: SQLTypeDate},

		// JSON and maps
		{name: "json", typeName: "json", expected: SQLTypeJSONB},
		{name: "jsonb", typeName: "jsonb", expected: SQLTypeJSONB},
		{name: "map_string_interface", typeName: "map[string]interface{}", expected: SQLTypeJSONB},
		{name: "map_string_any", typeName: "map[string]any", expected: SQLTypeJSONB},
		{name: "map_string_time", typeName: "map[string]time.Time", expected: SQLTypeJSONB},
		{name: "ptr_map", typeName: "*map[string]string", expected: SQLTypeJSONB},
		{name: "map", typeName: "map", expected: SQLTypeJSONB},

		// Byte slices
		{name: "bytes", typeName: "[]byte", expected: SQLTypeByte},

		// Pointer of a slice
		{name: "ptr_byte_slice", typeName: "*[]byte", expected: SQLTypeByte},
		{name: "ptr_string_slice", typeName: "*[]string", expected: SQLTypeArray},

		// Unmatched/Default
		{name: "custom_type", typeName: "github.com/example/package.CustomType", expected: SQLTypeText},
		{name: "empty_type", typeName: "", expected: SQLTypeText},
		{name: "unknown_type", typeName: "Foobar", expected: SQLTypeText},
		{name: "ptr_unknown", typeName: "*Foobar", expected: SQLTypeText},

		// Double pointer edge case
		{name: "double_ptr_int", typeName: "**int", expected: SQLTypeInteger}, // "**int" isn't realistically parsed but should default to TEXT
		// Weird casing
		{name: "all_caps", typeName: "INT", expected: SQLTypeInteger},
		{name: "mixed_caps", typeName: "InT", expected: SQLTypeInteger},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveSQLType(FieldInfo{
				Name:     tt.name,
				Type:     tt.typeName,
				Tag:      "",
				Embedded: false,
				Parent:   "",
			})
			if got != tt.expected {
				t.Errorf("ResolveSQLType(%q) = %v, want %v", tt.typeName, got, tt.expected)
			}
		})
	}
}

func TestPostgresColumnType(t *testing.T) {
	tests := []struct {
		typeName string
		want     string
	}{
		{typeName: "string", want: "TEXT"},
		{typeName: "[]string", want: "TEXT[]"},
		{typeName: "*[]string", want: "TEXT[]"},
		{typeName: "[]int", want: "INTEGER[]"},
		{typeName: "[]time.Time", want: "TIMESTAMP[]"},
		{typeName: "[]byte", want: "BYTEA"},
		{typeName: "map[string]any", want: "JSONB"},
		{typeName: "uuid.UUID", want: "UUID"},
	}
	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			got := PostgresColumnType(FieldInfo{Type: tt.typeName})
			if got != tt.want {
				t.Fatalf("PostgresColumnType(%q) = %q, want %q", tt.typeName, got, tt.want)
			}
		})
	}
}

func TestGenerateCreationSQL(t *testing.T) {
	p := parseModule(t, map[string]string{
		"user.go": `package parsertest

import (
	"time"

	"github.com/hunderaweke/berm/models"
)

type User struct {
	models.Model
	Name      string
	Tags      []string
	CreatedAt time.Time
}
`,
	})

	got := p.GenerateCreationSQL("user")
	wants := []string{
		`CREATE TABLE IF NOT EXISTS "user"`,
		`"id" UUID`,
		`"created_at" TIMESTAMP`,
		`"updated_at" TIMESTAMP`,
		`"name" TEXT`,
		`"tags" TEXT[]`,
		`PRIMARY KEY ("id")`,
	}
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("SQL missing %q\n%s", want, got)
		}
	}
	if strings.Count(got, `"created_at"`) != 1 {
		t.Fatalf("duplicate created_at columns:\n%s", got)
	}
	if strings.Contains(got, " ARRAY") || strings.Contains(got, "ARRAY,") {
		t.Fatalf("bare ARRAY is not valid PostgreSQL:\n%s", got)
	}
}

func BenchmarkResolveTableName(b *testing.B) {
	testNames := []string{
		"VeryLongName",
		"simple",
		"ID",
		"AnotherIDField",
		"FooBarBaz",
		"ABCDefG",
		"MyJSONField",
		"with1Number",
		"snake_case_will_stay",
		"already_snake_case",
		"__weird__case__",
		"TrailingNumbers123",
		"NameWith_MixOf_Cases",
	}

	fieldInfos := make([]FieldInfo, len(testNames))
	for i, n := range testNames {
		fieldInfos[i] = FieldInfo{
			Name:     n,
			Type:     "string",
			Tag:      "",
			Embedded: false,
			Parent:   "",
		}
	}
	b.ResetTimer()
	for b.Loop() {
		for _, fi := range fieldInfos {
			ResolveTableName(fi)
		}
	}
}

func BenchmarkResolveSQLType(b *testing.B) {
	typesToTest := []string{
		"string", "int", "int32", "int64", "uint64", "float32", "float64", "bool",
		"uuid.UUID", "UUID", "time.Time", "timestamp", "datetime", "date",
		"byte", "[]byte", "json", "jsonb", "map", "map[string]interface{}", "map[string]any",
		"[]string", "[]int", "[]float64", "array", "[]interface{}", "[]any", "[]bool",
		"github.com/example/package.CustomType",
		"", "Foobar", "*Foobar", "**int", "INT", "InT", "*[]byte", "*[]string",
	}

	// Also mix embedded fields and pointer variants
	fieldInfos := make([]FieldInfo, 0, len(typesToTest)*3)
	for _, typ := range typesToTest {
		fieldInfos = append(fieldInfos, FieldInfo{
			Name:     "TestField",
			Type:     typ,
			Tag:      "",
			Embedded: false,
			Parent:   "",
		})
		fieldInfos = append(fieldInfos, FieldInfo{
			Name:     "TestField",
			Type:     "*" + typ,
			Tag:      "",
			Embedded: false,
			Parent:   "",
		})
		fieldInfos = append(fieldInfos, FieldInfo{
			Name:     "TestField",
			Type:     typ,
			Tag:      "",
			Embedded: true,
			Parent:   "",
		})
	}

	b.ResetTimer()
	for b.Loop() {
		for _, fi := range fieldInfos {
			ResolveSQLType(fi)
		}
	}
}
