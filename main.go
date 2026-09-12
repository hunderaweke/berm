package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hunderaweke/berm/parser"
)

type Outer struct {
	Inner
	Something string
}

type Inner struct {
	ID string
}

func main() {
	src := `package testing

	   import (
	   	"time"

	   	"github.com/hunderaweke/berm/models"
	   )

	   type TestEmbeddedStruct struct{
	   	models.Model
	   	ID string
	   	Name string
	   	Number int
	   	CreatedAt time.Time
	   }`
	testPath := "../testing"
	err := os.MkdirAll(testPath, 0755)
	err = os.WriteFile(filepath.Join(testPath, "test.go"), []byte(src), 0644)
	if err != nil {
		log.Fatalf("failed writing test file: %v", err)
	}
	path, _ := filepath.Abs(testPath)
	log.Println("path:", path)
	cmd := exec.Command("go", "mod", "init", "test")
	cmd.Dir = path
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("failed to initialize module: %v, output: %s", err, string(output))
	}
	localMod, err := filepath.Abs(".")
	if err != nil {
		log.Fatalf("failed to resolve local module path: %v", err)
	}
	cmd = exec.Command("go", "mod", "edit", "-replace", "github.com/hunderaweke/berm="+localMod)
	cmd.Dir = path
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("failed to add replace directive: %v, output: %s", err, string(output))
	}
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = path
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("failed to tidy module: %v, output: %s", err, string(output))
	}
	fmt.Printf("module tidied successfully: %s", string(output))
	parser.ScanDir(testPath)
	os.RemoveAll(testPath)
}
