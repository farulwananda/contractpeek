package compare

import (
	"sort"
	"strings"

	"github.com/farulwananda/contractpeek/internal/baseline"
	"github.com/farulwananda/contractpeek/internal/shape"
)

// Severity indicates the breaking risk of an observed drift.
type Severity string

const (
	SeverityInfo              Severity = "info"
	SeverityPotentialBreaking Severity = "potential_breaking"
	SeverityBreaking          Severity = "breaking"
)

// ChangeKind describes the nature of the detected schema difference.
type ChangeKind string

const (
	KindFieldAdded            ChangeKind = "field_added"
	KindFieldNotObserved      ChangeKind = "field_not_observed"
	KindTypeChanged           ChangeKind = "type_changed"
	KindRequiredPathMissing   ChangeKind = "required_path_missing"
	KindArrayItemUnobservable ChangeKind = "array_item_unobservable"
)

// Change holds the details of a single field difference.
type Change struct {
	Path        string     `json:"path"`
	Kind        ChangeKind `json:"kind"`
	Severity    Severity   `json:"severity"`
	BeforeTypes []string   `json:"before_types"`
	AfterTypes  []string   `json:"after_types"`
	Reason      string     `json:"reason"`
}

// Summary aggregates finding counts by severity.
type Summary struct {
	Breaking          int `json:"breaking"`
	PotentialBreaking int `json:"potential_breaking"`
	Info              int `json:"info"`
}

// Result is the deterministic outcome of a contract comparison.
type Result struct {
	SchemaVersion int      `json:"schema_version"`
	ContractName  string   `json:"contract"`
	SampleCount   int      `json:"sample_count"`
	Summary       Summary  `json:"summary"`
	Changes       []Change `json:"changes"`
}

// Options controls comparison filtering and rules.
type Options struct {
	RequiredPaths []string
	IgnoredPaths  []string
}

// Compare evaluates an observed contract against an approved baseline.
func Compare(contractName string, base *baseline.Baseline, current *shape.InferredContract, opts Options) *Result {
	sampleCount := 0
	if current != nil {
		sampleCount = current.SampleCount
	}
	res := &Result{
		SchemaVersion: baseline.CurrentSchemaVersion,
		ContractName:  contractName,
		SampleCount:   sampleCount,
		Changes:       []Change{},
	}

	baseMap := make(map[string]baseline.Field)
	if base != nil {
		for _, f := range base.Fields {
			if !isIgnored(f.Path, opts.IgnoredPaths) {
				baseMap[f.Path] = f
			}
		}
	}

	currMap := make(map[string]shape.FieldInfo)
	if current != nil {
		for p, f := range current.Fields {
			if !isIgnored(p, opts.IgnoredPaths) {
				currMap[p] = *f
			}
		}
	}

	requiredSet := make(map[string]bool)
	for _, rp := range opts.RequiredPaths {
		requiredSet[rp] = true
	}

	// 1. Check all baseline fields against current observations
	for path, baseField := range baseMap {
		currField, exists := currMap[path]
		if !exists {
			// Field was in baseline, but absent in current sample
			if requiredSet[path] {
				res.Changes = append(res.Changes, Change{
					Path:        path,
					Kind:        KindRequiredPathMissing,
					Severity:    SeverityBreaking,
					BeforeTypes: baseField.Types,
					AfterTypes:  []string{},
					Reason:      "Declared required path was not observed in current response",
				})
			} else if isArrayItemDescendant(path, currMap) {
				// Parent array was empty in current sample -> unobservable, not removed
				res.Changes = append(res.Changes, Change{
					Path:        path,
					Kind:        KindArrayItemUnobservable,
					Severity:    SeverityInfo,
					BeforeTypes: baseField.Types,
					AfterTypes:  []string{},
					Reason:      "Enclosing array was empty in current sample; items cannot be observed",
				})
			} else {
				res.Changes = append(res.Changes, Change{
					Path:        path,
					Kind:        KindFieldNotObserved,
					Severity:    SeverityPotentialBreaking,
					BeforeTypes: baseField.Types,
					AfterTypes:  []string{},
					Reason:      "Field present in approved baseline was not observed in current response",
				})
			}
			continue
		}

		// Field exists in both: check type compatibility
		if !typeSetsMatch(baseField.Types, currField.Types) {
			res.Changes = append(res.Changes, Change{
				Path:        path,
				Kind:        KindTypeChanged,
				Severity:    SeverityPotentialBreaking,
				BeforeTypes: baseField.Types,
				AfterTypes:  currField.Types,
				Reason:      "Observed types differ from approved baseline",
			})
		}
	}
	// 2. Check declared required paths missing from current sample (even if not in baseline)
	for rp := range requiredSet {
		if isIgnored(rp, opts.IgnoredPaths) {
			continue
		}
		if _, inBase := baseMap[rp]; inBase {
			continue // Already evaluated in baseMap iteration
		}
		if _, inCurr := currMap[rp]; !inCurr {
			res.Changes = append(res.Changes, Change{
				Path:        rp,
				Kind:        KindRequiredPathMissing,
				Severity:    SeverityBreaking,
				BeforeTypes: []string{},
				AfterTypes:  []string{},
				Reason:      "Declared required path was not observed in current response",
			})
		}
	}

	// 3. Check newly added fields in current observations
	for path, currField := range currMap {
		if _, exists := baseMap[path]; !exists {
			res.Changes = append(res.Changes, Change{
				Path:        path,
				Kind:        KindFieldAdded,
				Severity:    SeverityInfo,
				BeforeTypes: []string{},
				AfterTypes:  currField.Types,
				Reason:      "New field observed that is not present in approved baseline",
			})
		}
	}

	// 3. Sort changes deterministically (Severity descending, then Path ascending)
	sort.Slice(res.Changes, func(i, j int) bool {
		sevOrder := map[Severity]int{
			SeverityBreaking:          3,
			SeverityPotentialBreaking: 2,
			SeverityInfo:              1,
		}
		if sevOrder[res.Changes[i].Severity] != sevOrder[res.Changes[j].Severity] {
			return sevOrder[res.Changes[i].Severity] > sevOrder[res.Changes[j].Severity]
		}
		return res.Changes[i].Path < res.Changes[j].Path
	})

	// 4. Compute summary counts
	for _, c := range res.Changes {
		switch c.Severity {
		case SeverityBreaking:
			res.Summary.Breaking++
		case SeverityPotentialBreaking:
			res.Summary.PotentialBreaking++
		case SeverityInfo:
			res.Summary.Info++
		}
	}

	return res
}

func isIgnored(path string, ignoredPaths []string) bool {
	for _, ip := range ignoredPaths {
		if shape.HasSubtreePrefix(path, ip) {
			return true
		}
	}
	return false
}

func isArrayItemDescendant(path string, currMap map[string]shape.FieldInfo) bool {
	lastStarIdx := strings.LastIndex(path, "/*")
	if lastStarIdx == -1 {
		return false
	}
	parentArrayPath := path[:lastStarIdx]
	itemPath := path[:lastStarIdx+2]

	// 1. Verify parent array was actually observed as an array in current sample
	arr, hasArray := currMap[parentArrayPath]
	if !hasArray {
		return false // Parent array itself was deleted/absent -> not merely an empty array
	}
	isArr := false
	for _, t := range arr.Types {
		if t == "array" {
			isArr = true
			break
		}
	}
	if !isArr {
		return false // Parent changed type away from array
	}

	// 2. If parent was observed as array, but items were unobserved (i.e. empty array `[]`),
	// then the item wildcard was unobservable
	_, hasItems := currMap[itemPath]
	return !hasItems
}

func typeSetsMatch(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]bool)
	for _, t := range a {
		set[t] = true
	}
	for _, t := range b {
		if !set[t] {
			return false
		}
	}
	return true
}
