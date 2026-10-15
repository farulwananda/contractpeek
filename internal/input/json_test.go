package input_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/farulwananda/contractpeek/internal/input"
)

func TestReadJSONFile_Valid(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "sample.json")
	content := `{"id": 12345, "name": "Alice", "tags": ["admin", "dev"], "meta": null}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	val, err := input.ReadJSONFile(path, input.Options{})
	if err != nil {
		t.Fatalf("expected valid JSON read, got error: %v", err)
	}

	m, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", val)
	}
	if m["name"] != "Alice" {
		t.Errorf("expected Alice, got %v", m["name"])
	}
}

func TestReadJSONFile_TrailingDataRejected(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "trailing.json")
	content := `{"id": 1} {"id": 2}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := input.ReadJSONFile(path, input.Options{})
	if err == nil {
		t.Fatal("expected error for trailing JSON data, got nil")
	}
}

func TestReadJSONFile_SizeLimitExceeded(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "oversized.json")
	content := `{"long_text": "012345678901234567890123456789"}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := input.ReadJSONFile(path, input.Options{MaxBytes: 15})
	if err == nil {
		t.Fatal("expected size limit error, got nil")
	}
}
