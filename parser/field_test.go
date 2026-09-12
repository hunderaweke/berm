package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func createEmbeddedStructAndTypeInfo() (string, error) {
	src := `package testing

	   import "github.com/hunderaweke/berm/models"

	   type TestEmbeddedStruct struct{
	   	models.Model
	   	ID string
	   	Name string
	   	Number int
	   	CreatedAt int
	   }`
	tmpDir := os.TempDir()
	srcFile := filepath.Join(tmpDir, "test.go")
	err := os.WriteFile(srcFile, []byte(src), 0644)
	if err != nil {
		return "", fmt.Errorf("failed writing test file: %v", err)
	}
	return tmpDir, nil
}

func TestIsTargetEmbeddedStruct(t *testing.T) {
	tmpDir, err := createEmbeddedStructAndTypeInfo()
	if err != nil {
		t.Fatalf("Failed creating embedded struct and type info: %v", err)
	}
	ScanDir(tmpDir)
}
