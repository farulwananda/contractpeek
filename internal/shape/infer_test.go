package shape_test

import (
	"encoding/json"
	"testing"

	"github.com/farulwananda/contractpeek/internal/shape"
)

func TestPath_Escaping(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normalKey", "normalKey"},
		{"key/with/slash", "key~1with~1slash"},
		{"key~with~tilde", "key~0with~0tilde"},
		{"literal*star", "literal~2star"},
		{"mixed/~/and/*", "mixed~1~0~1and~1~2"},
	}

	for _, tt := range tests {
		escaped := shape.EscapeSegment(tt.input)
		if escaped != tt.expected {
			t.Errorf("EscapeSegment(%q) = %q, want %q", tt.input, escaped, tt.expected)
		}
		unescaped := shape.UnescapeSegment(escaped)
		if unescaped != tt.input {
			t.Errorf("UnescapeSegment(%q) = %q, want %q", escaped, unescaped, tt.input)
		}
	}
}

func TestInferFromSamples_NestedObjectsAndArrays(t *testing.T) {
	sample1JSON := `{
		"data": {
			"user": {
				"id": "USR-101",
				"age": 28,
				"active": true
			},
			"items": [
				{"sku": "SKU-A", "qty": 2}
			]
		}
	}`

	sample2JSON := `{
		"data": {
			"user": {
				"id": "USR-102",
				"age": null,
				"active": false
			},
			"items": [
				{"sku": "SKU-B", "qty": 5}
			]
		}
	}`

	var s1, s2 any
	_ = json.Unmarshal([]byte(sample1JSON), &s1)
	_ = json.Unmarshal([]byte(sample2JSON), &s2)

	contract, err := shape.InferFromSamples([]any{s1, s2}, shape.InferOptions{})
	if err != nil {
		t.Fatalf("InferFromSamples failed: %v", err)
	}

	if contract.SampleCount != 2 {
		t.Errorf("expected 2 samples, got %d", contract.SampleCount)
	}

	// Verify path #/data/user/age has both number and null
	ageField := contract.Fields["#/data/user/age"]
	if ageField == nil {
		t.Fatal("expected #/data/user/age field, got nil")
	}
	if len(ageField.Types) != 2 || ageField.Types[0] != "null" || ageField.Types[1] != "number" {
		t.Errorf("expected [null, number], got %v", ageField.Types)
	}
	if ageField.NullInSamples != 1 {
		t.Errorf("expected NullInSamples 1, got %d", ageField.NullInSamples)
	}

	// Verify array item path #/data/items/*/sku
	skuField := contract.Fields["#/data/items/*/sku"]
	if skuField == nil {
		t.Fatal("expected #/data/items/*/sku field, got nil")
	}
	if len(skuField.Types) != 1 || skuField.Types[0] != "string" {
		t.Errorf("expected [string], got %v", skuField.Types)
	}
}
