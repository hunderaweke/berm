package parser

import (
	"fmt"
	"go/ast"
	"go/types"

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
								tableName := ResolveTableName(FieldInfo{
									Name: typeSpec.Name.Name,
								})
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
