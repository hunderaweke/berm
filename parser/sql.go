package parser

import (
	"regexp"
	"strings"
)

var (
	matchFirstCap = regexp.MustCompile("([a-z0-9])([A-Z])")
	matchAllCap   = regexp.MustCompile("([A-Z]+)([A-Z][a-z])")
)

func ResolveSQLType(info FieldInfo) string {
	typeName := strings.Split(info.Type, ".")[len(strings.Split(info.Type, "."))-1]
	switch typeName {
	case "UUID":
		return "UUID"
	case "Time":
		return "TIMESTAMP"
	default:
		return "TEXT"
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
