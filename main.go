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

type FieldInfo struct {
	Name     string
	Type     string
	Tag      string
	Embedded bool
	Parent   string
}

func ExtactAllFields(st *types.Struct, parentName string) []FieldInfo {
	var fields []FieldInfo
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		tag := st.Tag(i)
		fieldName := field.Name()
		fieldInfo := FieldInfo{
			Name:     fieldName,
			Type:     field.Type().String(),
			Tag:      tag,
			Embedded: field.Anonymous(),
			Parent:   parentName,
		}
		fields = append(fields, fieldInfo)
		if field.Anonymous() {
			typ := field.Type()
			if ptr, ok := typ.(*types.Pointer); ok {
				typ = ptr.Elem()
			}
			if embeddedStruct, ok := typ.Underlying().(*types.Struct); ok {
				innerFields := ExtactAllFields(embeddedStruct, field.Name())
				fields = append(fields, innerFields...)
			}
		}
	}
	return fields
}

func PrintStructDetails(typeSpec *ast.TypeSpec, info *types.Info) {
	obj := info.Defs[typeSpec.Name]
	if obj == nil {
		return
	}
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
						fmt.Printf("\n===== Struct:%s (%s:%d) =====\n", typeSpec.Name.Name, pkg.Dir, position.Line)
						obj := pkg.TypesInfo.Defs[typeSpec.Name]
						if obj != nil {
							if st, ok := obj.Type().Underlying().(*types.Struct); ok {
								allFields := ExtactAllFields(st, "")
								for _, f := range allFields {
									origin := ""
									if f.Parent != "" {
										origin = fmt.Sprintf(" (via %s)", f.Parent)
									}
									tagStr := ""
									if f.Tag != "" {
										tagStr = fmt.Sprintf(" `%s` ", f.Tag)
									}
									fmt.Printf("  - %-15s %-30s%s%s\n ", f.Name, f.Type, origin, tagStr)
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
