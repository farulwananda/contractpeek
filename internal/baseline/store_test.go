package baseline_test

import (
	"path/filepath"
	"testing"

	"github.com/farulwananda/contractpeek/internal/baseline"
	"github.com/farulwananda/contractpeek/internal/shape"
)

func TestStore_InitAndSave(t *testing.T) {
	tempDir := t.TempDir()
	store, err := baseline.NewStore(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify config.json was created
	cfg, err := store.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Defaults.FailOn != "breaking" {
		t.Errorf("expected default fail_on breaking, got %s", cfg.Defaults.FailOn)
	}

	// Create sample contract
	contract := &shape.InferredContract{
		SampleCount: 1,
		Fields: map[string]*shape.FieldInfo{
			"#/id":   {Path: "#/id", Types: []string{"number"}, ObservedInSamples: 1},
			"#/name": {Path: "#/name", Types: []string{"string"}, ObservedInSamples: 1},
		},
	}

	// Save new baseline
	if err := store.SaveBaseline("user-profile", contract, false); err != nil {
		t.Fatalf("SaveBaseline failed: %v", err)
	}

	// Attempting to overwrite without flag should fail
	if err := store.SaveBaseline("user-profile", contract, false); err == nil {
		t.Fatal("expected error on overwrite without flag, got nil")
	}

	// Loading baseline
	loaded, err := store.LoadBaseline("user-profile")
	if err != nil {
		t.Fatalf("LoadBaseline failed: %v", err)
	}
	if loaded.Name != "user-profile" || len(loaded.Fields) != 2 {
		t.Errorf("unexpected loaded baseline: %+v", loaded)
	}

	// List
	list, err := store.List()
	if err != nil || len(list) != 1 || list[0] != "user-profile" {
		t.Errorf("unexpected list: %v, %v", list, err)
	}
}

func TestStore_ValidateNamePathTraversal(t *testing.T) {
	invalidNames := []string{
		"../etc/passwd",
		"foo/bar",
		"foo\\bar",
		"name with spaces",
		"-leading-dash",
		"",
	}

	for _, n := range invalidNames {
		if err := baseline.ValidateName(n); err == nil {
			t.Errorf("expected error for invalid name %q, got nil", n)
		}
	}
}

func TestStore_LoadBaselineFile(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := baseline.NewStore(tempDir)
	contract := &shape.InferredContract{
		SampleCount: 1,
		Fields: map[string]*shape.FieldInfo{
			"#/status": {Path: "#/status", Types: []string{"string"}, ObservedInSamples: 1},
		},
	}
	_ = store.SaveBaseline("status", contract, true)

	file := filepath.Join(tempDir, ".contractpeek", "baselines", "status.json")
	b, err := baseline.LoadBaselineFile(file)
	if err != nil || b.Name != "status" {
		t.Fatalf("failed loading baseline file directly: %v", err)
	}
}
