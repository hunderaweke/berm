package parser

import (
	"regexp"
	"strings"
)

var (
	matchFirstCap = regexp.MustCompile("([a-z0-9])([A-Z])")
	matchAllCap   = regexp.MustCompile("([A-Z]+)([A-Z][a-z])")
)

type SQLType string

const (
	SQLTypeText      SQLType = "TEXT"
	SQLTypeInteger   SQLType = "INTEGER"
	SQLTypeReal      SQLType = "REAL"
	SQLTypeBoolean   SQLType = "BOOLEAN"
	SQLTypeUUID      SQLType = "UUID"
	SQLTypeTimestamp SQLType = "TIMESTAMP"
	SQLTypeDate      SQLType = "DATE"
	SQLTypeByte      SQLType = "BYTEA"
	SQLTypeJSONB     SQLType = "JSONB"
	SQLTypeArray     SQLType = "ARRAY"
	SQLTypeString    SQLType = "STRING"
	SQLTypeInt       SQLType = "INT"
	SQLTypeFloat     SQLType = "FLOAT"
	SQLTypeBool      SQLType = "BOOL"
	SQLTypeTime      SQLType = "TIME"
	SQLTypeDateTime  SQLType = "DATETIME"
)

func isMap(typeName string) bool {
	return strings.HasPrefix(typeName, "map")
}
func isArray(typeName string) bool {
	return strings.HasPrefix(typeName, "[]")
}
func isPointer(typeName string) bool {
	return strings.HasPrefix(typeName, "*")
}

func isInteger(typeName string) bool {
	matchInteger := regexp.MustCompile(`^(u?int.*)$`)
	f := matchInteger.Find([]byte(typeName))
	return f != nil
}
func ResolveSQLType(info FieldInfo) SQLType {
	typeName := strings.Split(info.Type, ".")[len(strings.Split(info.Type, "."))-1]
	typeName = strings.ToLower(strings.TrimPrefix(typeName, "*"))
	if isInteger(typeName) {
		return SQLTypeInteger
	}
	switch strings.ToLower(typeName) {
	case "string":
		return SQLTypeText
	case "float32", "float64":
		return SQLTypeReal
	case "bool":
		return SQLTypeBoolean
	case "uuid", "UUID":
		return SQLTypeUUID
	case "time", "timestamp", "datetime", "Time":
		return SQLTypeTimestamp
	case "date":
		return SQLTypeDate
	case "byte", "[]byte":
		return SQLTypeByte
	case "json", "jsonb", "map", "map[string]interface{}", "map[string]any":
		return SQLTypeJSONB
	case "[]string", "[]int", "[]float64", "array", "[]interface{}", "[]any", "[]bool":
		return SQLTypeArray
	default:
		if isMap(typeName) {
			return SQLTypeJSONB
		}
		if isArray(typeName) {
			return SQLTypeArray
		}
		if isPointer(typeName) {
			typeName = strings.TrimPrefix(typeName, "*")
			return ResolveSQLType(FieldInfo{Type: typeName})
		}
		return SQLTypeText
	}
}

func toSnakeCase(str string) string {
	str = matchAllCap.ReplaceAllString(str, "${1}_${2}")
	str = matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	return strings.ToLower(str)
}

func ResolveTableName(info FieldInfo) string {
	if info.Embedded {
		return ""
	}
	return toSnakeCase(info.Name)
}
