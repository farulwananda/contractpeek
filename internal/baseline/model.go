package baseline

import "time"

// CurrentSchemaVersion is the stable schema format version for baselines and configs.
const CurrentSchemaVersion = 1

// Field defines the observed types and occurrences for a canonical pointer path.
type Field struct {
	Path              string   `json:"path"`
	Types             []string `json:"types"`
	ObservedInSamples int      `json:"observed_in_samples"`
	NullInSamples     int      `json:"null_in_samples"`
}

// Baseline represents an approved API contract baseline document.
type Baseline struct {
	SchemaVersion int       `json:"schema_version"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"created_at"`
	SampleCount   int       `json:"sample_count"`
	Fields        []Field   `json:"fields"`
}

// ContractRule defines declared expectations for a named contract in config.json.
type ContractRule struct {
	RequiredPaths []string `json:"required_paths,omitempty"`
	IgnoredPaths  []string `json:"ignored_paths,omitempty"`
}

// ConfigDefaults specifies project-wide defaults.
type ConfigDefaults struct {
	FailOn        string `json:"fail_on"`
	MaxInputBytes int64  `json:"max_input_bytes"`
	MaxDepth      int    `json:"max_depth"`
}

// Config represents .contractpeek/config.json.
type Config struct {
	SchemaVersion int                     `json:"schema_version"`
	Defaults      ConfigDefaults          `json:"defaults"`
	Contracts     map[string]ContractRule `json:"contracts"`
}
