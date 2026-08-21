package parser

import (
	"strings"
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
