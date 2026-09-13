package parser

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

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
