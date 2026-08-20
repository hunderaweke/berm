package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"path/filepath"
)

type Outer struct {
	Inner
	Something string
}

type Inner struct {
	ID string
}

const targetStruct = "Inner"
const targetDir = "."

func main() {
	fset := token.NewFileSet()

	err := filepath.WalkDir(targetDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			scanDir(fset, path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("error walking through the project: %v", err)
	}
}
func scanDir(fset *token.FileSet, dirPath string) {
	pkgs, err := parser.ParseDir(fset, dirPath, nil, parser.AllErrors)
	if err != nil {
		log.Fatalf("error parsing the directory: %v", err)
	}
	for filename, pkg := range pkgs {
		log.Printf("Package: %s", pkg.Name)
		for _, file := range pkg.Files {
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
					if isTargetField(field.Type, "Model") {
						position := fset.Position(typeSpec.Pos())
						fmt.Printf("Found: struct '%s' in %s:%d\n", typeSpec.Name.Name, filename, position.Line)
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
