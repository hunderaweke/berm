package parser

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/google/uuid"

	"golang.org/x/tools/go/packages"
)

type StructInfo struct {
	Name      string
	TableName string
	Fields    []FieldInfo
}

type Parser struct {
	Dir     string
	Structs map[string]StructInfo
}

func NewParser(dir string) *Parser {
	return &Parser{
		Dir:     dir,
		Structs: make(map[string]StructInfo),
	}
}

func (p *Parser) Parse() error {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
		Dir: p.Dir,
	}
	pkgs, err := packages.Load(cfg, p.Dir)
	if err != nil {
		return fmt.Errorf("error parsing the directory: %v", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		return fmt.Errorf("type errors while scanning %s", p.Dir)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				typeSpec, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					return true
				}
				for _, field := range structType.Fields.List {
					if IsTargetEmbeddedStruct(field, pkg.TypesInfo, "github.com/hunderaweke/berm/models", "Model") {
						obj := pkg.TypesInfo.Defs[typeSpec.Name]
						if obj != nil {
							if st, ok := obj.Type().Underlying().(*types.Struct); ok {
								allFields := ExtactAllFields(st, "")
								tableName := fmt.Sprintf("%ss", ResolveTableName(FieldInfo{
									Name: typeSpec.Name.Name,
								}))
								p.Structs[tableName] = StructInfo{
									Name:      typeSpec.Name.Name,
									TableName: tableName,
									Fields:    allFields,
								}
							}
						}
						return true
					}
				}
				return false
			})
		}
	}
	return nil
}

func (p *Parser) GenerateCreationSQL(tableName string) string {
	info, ok := p.Structs[tableName]
	if !ok {
		return ""
	}
	seen := make(map[string]bool)
	cols := make([]string, 0, len(info.Fields)+1)
	for _, field := range info.Fields {
		fieldName := ResolveTableName(field)
		if fieldName == "" || seen[fieldName] {
			continue
		}
		seen[fieldName] = true
		cols = append(cols, fmt.Sprintf("%s %s", quoteIdent(fieldName), PostgresColumnType(field)))
	}
	return fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (%s, PRIMARY KEY (%s));\n",
		quoteIdent(tableName),
		strings.Join(cols, ", "),
		quoteIdent("id"),
	)
}

func (p *Parser) GenerateCreationSQLForAllTables() string {
	sql := ""
	for tableName := range p.Structs {
		sql += p.GenerateCreationSQL(tableName) + "\n"
	}
	return sql
}

func insertColumnNames(info StructInfo, rows []map[string]any) []string {
	seen := make(map[string]bool, len(info.Fields))
	cols := make([]string, 0, len(info.Fields))
	for _, field := range info.Fields {
		name := ResolveTableName(field)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		if name == "id" {
			cols = append(cols, name)
			continue
		}
		for _, row := range rows {
			if row[name] != nil {
				cols = append(cols, name)
				break
			}
		}
	}
	return cols
}

func (p *Parser) GenerateInsertSQL(tableName string, data map[string]any) string {
	if data == nil {
		return ""
	}
	return p.GenerateBatchInsertSQL(tableName, []map[string]any{data})
}

func (p *Parser) GenerateBatchInsertSQL(tableName string, rows []map[string]any) string {
	info, ok := p.Structs[tableName]
	if !ok || len(rows) == 0 {
		return ""
	}
	cols := insertColumnNames(info, rows)
	if len(cols) == 0 {
		return ""
	}

	var b strings.Builder
	b.Grow(48 + len(tableName) + len(cols)*24 + len(rows)*(len(cols)*16+4))
	b.WriteString("INSERT INTO ")
	b.WriteString(quoteIdent(tableName))
	b.WriteString(" (")
	for i, col := range cols {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(quoteIdent(col))
	}
	b.WriteString(") VALUES ")
	for i, row := range rows {
		if row == nil {
			row = map[string]any{}
		}
		if row["id"] == nil {
			row["id"] = uuid.New().String()
		}
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteByte('(')
		for j, col := range cols {
			if j > 0 {
				b.WriteString(", ")
			}
			b.WriteString(quoteSQLValue(row[col]))
		}
		b.WriteByte(')')
	}
	b.WriteByte(';')
	return b.String()
}

func (p *Parser) GenerateSelectSQL(tableName string, columns []string) string {
	_, ok := p.Structs[tableName]
	if !ok {
		return ""
	}
	if len(columns) == 0 {
		columns = append(columns, "*")
	}
	var b strings.Builder
	b.WriteString("SELECT ")
	for i, col := range columns {
		b.WriteString(col)
		if i != len(columns)-1 {
			b.WriteString(", ")
		}
	}
	b.WriteString(fmt.Sprintf("FROM '%s';", tableName))
	return b.String()
}
