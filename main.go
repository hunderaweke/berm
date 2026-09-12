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

	"github.com/google/uuid"
	"github.com/hunderaweke/berm/models"
)

type Address struct {
	Street  string
	City    string
	ZipCode string
	Geo     map[string]float64
}

type Profile struct {
	Bio      string
	Website  string
	Birthday time.Time
	Socials  map[string]string
}

type Audit struct {
	DeletedAt *time.Time
	Version   int
}

type FirstStruct struct {
	models.Model
	Name      string
	Number    int
	CreatedAt time.Time
}

type SecondStruct struct {
	models.Model
	Audit
	Title       string
	Active      bool
	Score       float64
	Tags        []string
	Metadata    map[string]any
	Settings    map[string]string
	Avatar      []byte
	UserID      uuid.UUID
	PublishedAt *time.Time
	Address     Address
	Profile     *Profile
	Addresses   []Address
}
`
	testPath := "../testing"
	err := os.MkdirAll(testPath, 0755)
	err = os.WriteFile(filepath.Join(testPath, "test.go"), []byte(src), 0644)
	if err != nil {
		log.Fatalf("failed writing test file: %v", err)
	}
	path, _ := filepath.Abs(testPath)
	if _, err := os.Stat(filepath.Join(path, "go.mod")); os.IsNotExist(err) {
		cmd := exec.Command("go", "mod", "init", "test")
		cmd.Dir = path
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("failed to initialize module: %v, output: %s", err, string(output))
		}
	}
	localMod, err := filepath.Abs(".")
	if err != nil {
		log.Fatalf("failed to resolve local module path: %v", err)
	}
	cmd := exec.Command("go", "mod", "edit", "-replace", "github.com/hunderaweke/berm="+localMod)
	cmd.Dir = path
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("failed to add replace directive: %v, output: %s", err, string(output))
	}
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = path
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("failed to tidy module: %v, output: %s", err, string(output))
	}
	p := parser.NewParser(path)
	err = p.Parse()
	if err != nil {
		log.Fatalf("Failed parsing: %v", err)
	}
	for name, fieldInfos := range p.Structs {
		fmt.Printf("\n===== %s =====\n", name)
		for _, fieldInfo := range fieldInfos.Fields {
			fmt.Printf("  - %-15s %-12s %-12s %s\n",
				fieldInfo.Name,
				parser.ResolveSQLType(fieldInfo),
				fieldInfo.Type,
				fieldInfo.Tag,
			)
		}
	}
	os.RemoveAll(testPath)
}
