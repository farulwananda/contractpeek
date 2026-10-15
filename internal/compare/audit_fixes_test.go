package compare_test

import (
	"testing"

	"github.com/farulwananda/contractpeek/internal/baseline"
	"github.com/farulwananda/contractpeek/internal/compare"
	"github.com/farulwananda/contractpeek/internal/shape"
)

func TestCompare_DeclaredRequiredPathNotInBaseline(t *testing.T) {
	// Baseline was captured without #/data/token
	base := &baseline.Baseline{
		Name: "auth-api",
		Fields: []baseline.Field{
			{Path: "#/data/user", Types: []string{"string"}},
		},
	}

	// Current response also lacks #/data/token
	curr := &shape.InferredContract{
		SampleCount: 1,
		Fields: map[string]*shape.FieldInfo{
			"#/data/user": {Path: "#/data/user", Types: []string{"string"}},
		},
	}

	// User config declares #/data/token as mandatory
	opts := compare.Options{
		RequiredPaths: []string{"#/data/token"},
	}

	res := compare.Compare("auth-api", base, curr, opts)
	if res.Summary.Breaking != 1 {
		t.Fatalf("expected 1 breaking change for declared required path, got %d", res.Summary.Breaking)
	}
	if len(res.Changes) != 1 || res.Changes[0].Path != "#/data/token" {
		t.Fatalf("expected finding on #/data/token, got: %+v", res.Changes)
	}
}

func TestCompare_PopulatedArrayMissingItemFieldIsPotentialBreaking(t *testing.T) {
	// Baseline had array with items having both sku and qty
	base := &baseline.Baseline{
		Name: "cart-api",
		Fields: []baseline.Field{
			{Path: "#/items", Types: []string{"array"}},
			{Path: "#/items/*", Types: []string{"object"}},
			{Path: "#/items/*/sku", Types: []string{"string"}},
			{Path: "#/items/*/qty", Types: []string{"number"}},
		},
	}

	// Current response has populated array, but items are missing qty
	curr := &shape.InferredContract{
		SampleCount: 1,
		Fields: map[string]*shape.FieldInfo{
			"#/items":       {Path: "#/items", Types: []string{"array"}},
			"#/items/*":     {Path: "#/items/*", Types: []string{"object"}},
			"#/items/*/sku": {Path: "#/items/*/sku", Types: []string{"string"}},
		},
	}

	res := compare.Compare("cart-api", base, curr, compare.Options{})
	if res.Summary.PotentialBreaking != 1 {
		t.Fatalf("expected 1 potential breaking change for missing qty in populated array, got %d", res.Summary.PotentialBreaking)
	}
	if res.Changes[0].Kind != compare.KindFieldNotObserved || res.Changes[0].Path != "#/items/*/qty" {
		t.Fatalf("expected KindFieldNotObserved for #/items/*/qty, got: %+v", res.Changes[0])
	}
}
