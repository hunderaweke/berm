package parser

import "testing"

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
		expected string
	}{
		{name: "username", typeName: "string", expected: "TEXT"},
		{name: "id", typeName: "uuid.UUID", expected: "UUID"},
		{name: "createdAt", typeName: "time.Time", expected: "TIMESTAMP"},
		{name: "updatedAt", typeName: "time.Time", expected: "TIMESTAMP"},
		{name: "name", typeName: "string", expected: "TEXT"},
		{name: "id", typeName: "uuid.UUID", expected: "UUID"},
		{name: "createdAt", typeName: "time.Time", expected: "TIMESTAMP"},
		{name: "updatedAt", typeName: "time.Time", expected: "TIMESTAMP"},
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
				t.Errorf("ResolveSQLType() = %v, want %v", got, tt.expected)
			}
		})
	}
}
func BenchmarkResolveTableName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ResolveTableName(FieldInfo{
			Name:     "VeryLongName",
			Type:     "string",
			Tag:      "",
			Embedded: false,
			Parent:   "",
		})
	}
}

func BenchmarkResolveSQLType(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ResolveSQLType(FieldInfo{
			Name:     "VeryLongName",
			Type:     "string",
			Tag:      "",
			Embedded: false,
			Parent:   "",
		})
	}
}
