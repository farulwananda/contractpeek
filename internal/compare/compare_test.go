package compare_test

import (
	"testing"

	"github.com/farulwananda/contractpeek/internal/baseline"
	"github.com/farulwananda/contractpeek/internal/compare"
	"github.com/farulwananda/contractpeek/internal/shape"
)

func TestCompare_BreakingMissingRequiredPath(t *testing.T) {
	base := &baseline.Baseline{
		Name: "test-api",
		Fields: []baseline.Field{
			{Path: "#/id", Types: []string{"string"}},
			{Path: "#/status", Types: []string{"string"}},
		},
	}

	curr := &shape.InferredContract{
		Fields: map[string]*shape.FieldInfo{
			"#/status": {Path: "#/status", Types: []string{"string"}},
		},
	}

	opts := compare.Options{
		RequiredPaths: []string{"#/id"},
	}

	res := compare.Compare("test-api", base, curr, opts)

	if res.Summary.Breaking != 1 {
		t.Errorf("expected 1 breaking change, got %d", res.Summary.Breaking)
	}
	if len(res.Changes) != 1 || res.Changes[0].Kind != compare.KindRequiredPathMissing {
		t.Fatalf("unexpected changes: %+v", res.Changes)
	}
}

func TestCompare_TypeChanged(t *testing.T) {
	base := &baseline.Baseline{
		Name: "test-api",
		Fields: []baseline.Field{
			{Path: "#/count", Types: []string{"number"}},
		},
	}

	curr := &shape.InferredContract{
		Fields: map[string]*shape.FieldInfo{
			"#/count": {Path: "#/count", Types: []string{"string"}},
		},
	}

	res := compare.Compare("test-api", base, curr, compare.Options{})

	if res.Summary.PotentialBreaking != 1 {
		t.Errorf("expected 1 potential breaking change, got %d", res.Summary.PotentialBreaking)
	}
	if res.Changes[0].Kind != compare.KindTypeChanged {
		t.Errorf("expected KindTypeChanged, got %s", res.Changes[0].Kind)
	}
}

func TestCompare_IgnoredPaths(t *testing.T) {
	base := &baseline.Baseline{
		Name: "test-api",
		Fields: []baseline.Field{
			{Path: "#/meta/debug", Types: []string{"string"}},
			{Path: "#/data", Types: []string{"object"}},
		},
	}

	curr := &shape.InferredContract{
		Fields: map[string]*shape.FieldInfo{
			"#/data": {Path: "#/data", Types: []string{"object"}},
		},
	}

	opts := compare.Options{
		IgnoredPaths: []string{"#/meta"},
	}

	res := compare.Compare("test-api", base, curr, opts)
	if len(res.Changes) != 0 {
		t.Errorf("expected 0 changes due to ignored path, got %d: %+v", len(res.Changes), res.Changes)
	}
}
