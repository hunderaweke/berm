package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"log"

	"github.com/hunderaweke/berm/parser"
	"golang.org/x/tools/go/packages"
)

type Outer struct {
	Inner
	Something string
}

type Inner struct {
	ID string
}

func main() {
	scanDir()
}

func scanDir() {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
		Dir: ".",
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		log.Fatalf("error parsing the directory: %v", err)
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
					if parser.IsTargetEmbeddedStruct(field, pkg.TypesInfo, "github.com/hunderaweke/berm/models", "Model") {
						position := pkg.Fset.Position(typeSpec.Pos())
						fmt.Printf("\n===== Struct:%s (%s:%d) =====\n", typeSpec.Name.Name, pkg.Dir, position.Line)
						obj := pkg.TypesInfo.Defs[typeSpec.Name]
						if obj != nil {
							if st, ok := obj.Type().Underlying().(*types.Struct); ok {
								allFields := parser.ExtactAllFields(st, "")
								for _, f := range allFields {
									sqlType := parser.ResolveSQLType(f)
									tableName := parser.ResolveTableName(f)
									fmt.Printf("  - %-15s %-30s\n", tableName, sqlType)
								}
							}
						}
						// PrintStructDetails(typeSpec, pkg.TypesInfo)
						return true
					}
				}
				return false
			})
		}
	}
}
func isTargetField(expr ast.Expr, target string) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name == target
	case *ast.StarExpr:
		return isTargetField(t.X, target)
	case *ast.SelectorExpr:
		return t.Sel.Name == target
	default:
		return false
	}
}
