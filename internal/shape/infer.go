package shape

import (
	"encoding/json"
	"fmt"
	"sort"
)

// FieldInfo describes the observed types and occurrences of a JSON path.
type FieldInfo struct {
	Path              string   `json:"path"`
	Types             []string `json:"types"`
	ObservedInSamples int      `json:"observed_in_samples"`
	NullInSamples     int      `json:"null_in_samples"`
}

// InferredContract represents the normalized aggregate shape inferred from one or more sample responses.
type InferredContract struct {
	SampleCount int                   `json:"sample_count"`
	Fields      map[string]*FieldInfo `json:"fields"`
}

// InferOptions controls recursion limits during shape inference.
type InferOptions struct {
	MaxDepth int
}

// InferFromSamples analyzes one or more parsed JSON values and returns a unified InferredContract.
func InferFromSamples(samples []any, opts InferOptions) (*InferredContract, error) {
	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 64
	}

	contract := &InferredContract{
		SampleCount: len(samples),
		Fields:      make(map[string]*FieldInfo),
	}

	for _, sample := range samples {
		samplePaths := make(map[string]map[string]bool)
		sampleNulls := make(map[string]bool)

		if err := walk(sample, "#", 0, maxDepth, samplePaths, sampleNulls); err != nil {
			return nil, err
		}

		// Merge sample occurrences into master contract
		for path, typeSet := range samplePaths {
			info, exists := contract.Fields[path]
			if !exists {
				info = &FieldInfo{
					Path:  path,
					Types: []string{},
				}
				contract.Fields[path] = info
			}

			info.ObservedInSamples++
			if sampleNulls[path] {
				info.NullInSamples++
			}

			// Add distinct types
			existingTypes := make(map[string]bool)
			for _, t := range info.Types {
				existingTypes[t] = true
			}
			for t := range typeSet {
				if !existingTypes[t] {
					info.Types = append(info.Types, t)
				}
			}
			sort.Strings(info.Types)
		}
	}

	return contract, nil
}

// SortedFields returns all fields in deterministic lexicographical order.
func (c *InferredContract) SortedFields() []FieldInfo {
	var paths []string
	for p := range c.Fields {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	res := make([]FieldInfo, len(paths))
	for i, p := range paths {
		res[i] = *c.Fields[p]
	}
	return res
}

func walk(val any, path string, depth, maxDepth int, paths map[string]map[string]bool, nulls map[string]bool) error {
	if depth > maxDepth {
		return fmt.Errorf("nesting depth exceeded maximum limit of %d at path %s", maxDepth, path)
	}

	if paths[path] == nil {
		paths[path] = make(map[string]bool)
	}

	if val == nil {
		paths[path]["null"] = true
		nulls[path] = true
		return nil
	}

	switch v := val.(type) {
	case map[string]any:
		paths[path]["object"] = true
		for k, childVal := range v {
			childPath := JoinPath(path, k)
			if err := walk(childVal, childPath, depth+1, maxDepth, paths, nulls); err != nil {
				return err
			}
		}

	case []any:
		paths[path]["array"] = true
		itemPath := JoinArrayItem(path)
		for _, item := range v {
			if err := walk(item, itemPath, depth+1, maxDepth, paths, nulls); err != nil {
				return err
			}
		}

	case string:
		paths[path]["string"] = true

	case json.Number, float64, int, int64:
		paths[path]["number"] = true

	case bool:
		paths[path]["boolean"] = true

	default:
		paths[path]["unknown"] = true
	}

	return nil
}
