package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"

	"golang.org/x/tools/go/packages"
)

type Outer struct {
	Inner
	Something string
}

type Inner struct {
	ID string
}

func PrintStructDetails(typeSpec *ast.TypeSpec, info *types.Info) {
	// Look up the type definition object
	obj := info.Defs[typeSpec.Name]
	if obj == nil {
		return
	}

	// Resolve the underlying struct type
	structType, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		return
	}

	fmt.Printf("  Fields (%d total):\n", structType.NumFields())
	for i := 0; i < structType.NumFields(); i++ {
		fieldVar := structType.Field(i)
		tag := structType.Tag(i)

		embeddedStr := ""
		if fieldVar.Anonymous() {
			embeddedStr = " [Embedded]"
		}

		tagStr := ""
		if tag != "" {
			tagStr = fmt.Sprintf(" `%s`", tag)
		}

		fmt.Printf("    - Name: %-15s Type: %-25s%s%s\n",
			fieldVar.Name(),
			fieldVar.Type().String(),
			embeddedStr,
			tagStr,
		)
	}
}
func IsTargetEmbeddedStruct(field *ast.Field, info *types.Info, targetPathPkg, targetStructName string) bool {

	if len(field.Names) != 0 {
		return false
	}
	if info == nil {
		return false
	}

	tv, ok := info.Types[field.Type]
	if !ok {
		return false
	}

	typ := tv.Type
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}

	named, ok := typ.(*types.Named)
	if !ok {
		return false
	}

	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return obj.Pkg().Path() == targetPathPkg && obj.Name() == targetStructName
}

const targetStruct = "Model"
const targetDir = "."

func main() {
	fset := token.NewFileSet()
	scanDir(fset)
}
func scanDir(fset *token.FileSet) {
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
					if IsTargetEmbeddedStruct(field, pkg.TypesInfo, "struct-inheritance/models", "Model") {
						position := pkg.Fset.Position(typeSpec.Pos())
						fmt.Printf("Found: struct '%s' in %s:%d\n", typeSpec.Name.Name, pkg.Dir, position.Line)
						PrintStructDetails(typeSpec, pkg.TypesInfo)
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
