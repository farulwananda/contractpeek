package baseline

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/farulwananda/contractpeek/internal/shape"
)

var validNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// ValidateName ensures baseline names prevent path traversal and follow strict alphanumeric conventions.
func ValidateName(name string) error {
	if name == "" {
		return errors.New("contract baseline name cannot be empty")
	}
	if !validNameRegex.MatchString(name) {
		return fmt.Errorf("invalid contract baseline name %q: must match ^[a-z0-9][a-z0-9_-]{0,63}$", name)
	}
	return nil
}

// Store manages the local .contractpeek/ directory structure.
type Store struct {
	projectDir  string
	contractDir string
	baselineDir string
	configFile  string
}

// NewStore initializes a Store in projectDir (defaults to current working directory).
func NewStore(projectDir string) (*Store, error) {
	if projectDir == "" {
		projectDir = "."
	}
	contractDir := filepath.Join(projectDir, ".contractpeek")
	baselineDir := filepath.Join(contractDir, "baselines")
	configFile := filepath.Join(contractDir, "config.json")

	return &Store{
		projectDir:  projectDir,
		contractDir: contractDir,
		baselineDir: baselineDir,
		configFile:  configFile,
	}, nil
}

// Init creates .contractpeek and config.json if not already present.
func (s *Store) Init() error {
	if err := os.MkdirAll(s.baselineDir, 0755); err != nil {
		return fmt.Errorf("failed creating baseline directory: %w", err)
	}

	if _, err := os.Stat(s.configFile); errors.Is(err, os.ErrNotExist) {
		defaultCfg := Config{
			SchemaVersion: CurrentSchemaVersion,
			Defaults: ConfigDefaults{
				FailOn:        "breaking",
				MaxInputBytes: 10 * 1024 * 1024,
				MaxDepth:      64,
			},
			Contracts: make(map[string]ContractRule),
		}
		data, _ := json.MarshalIndent(defaultCfg, "", "  ")
		if err := os.WriteFile(s.configFile, data, 0644); err != nil {
			return fmt.Errorf("failed writing config.json: %w", err)
		}
	}

	return nil
}

// Exists checks if a named baseline exists.
func (s *Store) Exists(name string) bool {
	if err := ValidateName(name); err != nil {
		return false
	}
	path := filepath.Join(s.baselineDir, strings.ToLower(name)+".json")
	_, err := os.Stat(path)
	return err == nil
}

// SaveBaseline writes an InferredContract as a named baseline atomically.
// If overwrite is false and the baseline already exists, it returns an error.
func (s *Store) SaveBaseline(name string, contract *shape.InferredContract, overwrite bool) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	name = strings.ToLower(name)

	if !overwrite && s.Exists(name) {
		return fmt.Errorf("contract baseline %q already exists; use 'baseline update --name %s --approve' to update", name, name)
	}

	if err := os.MkdirAll(s.baselineDir, 0755); err != nil {
		return fmt.Errorf("failed creating baseline directory: %w", err)
	}

	sorted := contract.SortedFields()
	fields := make([]Field, len(sorted))
	for i, f := range sorted {
		fields[i] = Field{
			Path:              f.Path,
			Types:             f.Types,
			ObservedInSamples: f.ObservedInSamples,
			NullInSamples:     f.NullInSamples,
		}
	}

	b := Baseline{
		SchemaVersion: CurrentSchemaVersion,
		Name:          name,
		CreatedAt:     time.Now().UTC().Truncate(time.Second),
		SampleCount:   contract.SampleCount,
		Fields:        fields,
	}

	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("failed encoding baseline JSON: %w", err)
	}

	finalPath := filepath.Join(s.baselineDir, name+".json")
	tmpFile, err := os.CreateTemp(s.baselineDir, fmt.Sprintf(".tmp_%s_*", name))
	if err != nil {
		return fmt.Errorf("failed creating temp baseline file: %w", err)
	}
	tmpPath := tmpFile.Name()

	success := false
	defer func() {
		if !success {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	_ = tmpFile.Chmod(0644)
	if _, err := tmpFile.Write(data); err != nil {
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("failed replacing baseline file: %w", err)
	}

	success = true
	return nil
}

// LoadBaseline loads and validates a named baseline from disk.
func (s *Store) LoadBaseline(name string) (*Baseline, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	path := filepath.Join(s.baselineDir, strings.ToLower(name)+".json")
	return LoadBaselineFile(path)
}

// LoadBaselineFile loads a baseline document directly from an arbitrary path.
func LoadBaselineFile(path string) (*Baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read baseline file %q: %w", path, err)
	}

	var b Baseline
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("invalid baseline JSON in %q: %w", path, err)
	}

	if b.SchemaVersion != CurrentSchemaVersion {
		return nil, fmt.Errorf("unsupported baseline schema_version %d in %q (supported: %d)",
			b.SchemaVersion, path, CurrentSchemaVersion)
	}

	return &b, nil
}

// LoadConfig loads project configuration.
func (s *Store) LoadConfig() (*Config, error) {
	data, err := os.ReadFile(s.configFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{
				SchemaVersion: CurrentSchemaVersion,
				Defaults: ConfigDefaults{
					FailOn:        "breaking",
					MaxInputBytes: 10 * 1024 * 1024,
					MaxDepth:      64,
				},
				Contracts: make(map[string]ContractRule),
			}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config.json in %q: %w", s.configFile, err)
	}

	if cfg.SchemaVersion != CurrentSchemaVersion {
		return nil, fmt.Errorf("unsupported config schema_version %d in %q (supported: %d)",
			cfg.SchemaVersion, s.configFile, CurrentSchemaVersion)
	}
	if cfg.Defaults.FailOn == "" {
		cfg.Defaults.FailOn = "breaking"
	}
	if cfg.Defaults.MaxInputBytes <= 0 {
		cfg.Defaults.MaxInputBytes = 10 * 1024 * 1024
	}
	if cfg.Defaults.MaxDepth <= 0 {
		cfg.Defaults.MaxDepth = 64
	}

	return &cfg, nil
}

// List returns a list of all existing baseline names.
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.baselineDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") && !strings.HasPrefix(entry.Name(), ".") {
			name := strings.TrimSuffix(entry.Name(), ".json")
			if err := ValidateName(name); err == nil {
				names = append(names, name)
			}
		}
	}
	return names, nil
}
