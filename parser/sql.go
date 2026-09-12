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

func stripPointers(typeName string) string {
	for strings.HasPrefix(typeName, "*") {
		typeName = typeName[1:]
	}
	return typeName
}

func baseTypeName(typeName string) string {
	if i := strings.LastIndex(typeName, "."); i >= 0 {
		return typeName[i+1:]
	}
	return typeName
}

func isMap(typeName string) bool {
	return typeName == "map" || strings.HasPrefix(typeName, "map[")
}

func isArray(typeName string) bool {
	return strings.HasPrefix(typeName, "[]")
}

func isByte(typeName string) bool {
	switch typeName {
	case "byte", "[]byte", "bytes", "bytea":
		return true
	default:
		return false
	}
}

func isInteger(typeName string) bool {
	switch typeName {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"integer":
		return true
	default:
		return false
	}
}

func isFloat(typeName string) bool {
	switch typeName {
	case "float", "float32", "float64", "real":
		return true
	default:
		return false
	}
}

func isBool(typeName string) bool {
	return typeName == "bool" || typeName == "boolean"
}

func isUUID(typeName string) bool {
	return typeName == "uuid"
}

func isTime(typeName string) bool {
	switch typeName {
	case "time", "timestamp", "datetime":
		return true
	default:
		return false
	}
}

func isDate(typeName string) bool {
	return typeName == "date"
}

func isJSON(typeName string) bool {
	switch typeName {
	case "json", "jsonb", "rawmessage":
		return true
	default:
		return false
	}
}

func ResolveSQLType(info FieldInfo) SQLType {
	typeName := strings.ToLower(stripPointers(strings.TrimSpace(info.Type)))

	// Structural types must be matched on the full type, before taking
	// the last identifier. Otherwise map[string]time.Time becomes "time".
	if isByte(typeName) {
		return SQLTypeByte
	}
	if isMap(typeName) {
		return SQLTypeJSONB
	}
	if isArray(typeName) {
		return SQLTypeArray
	}

	typeName = baseTypeName(typeName)
	switch {
	case isInteger(typeName):
		return SQLTypeInteger
	case isFloat(typeName):
		return SQLTypeReal
	case isBool(typeName):
		return SQLTypeBoolean
	case isUUID(typeName):
		return SQLTypeUUID
	case isTime(typeName):
		return SQLTypeTimestamp
	case isDate(typeName):
		return SQLTypeDate
	case isByte(typeName):
		return SQLTypeByte
	case isJSON(typeName):
		return SQLTypeJSONB
	default:
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
