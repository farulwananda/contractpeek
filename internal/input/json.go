package input

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	// DefaultMaxInputBytes is the standard 10 MiB limit per sample file.
	DefaultMaxInputBytes = 10 * 1024 * 1024
	// DefaultMaxDepth is the recursion limit to prevent stack exhaustion on deeply nested malicious JSON.
	DefaultMaxDepth = 64
)

// Options specifies limits for reading JSON input.
type Options struct {
	MaxBytes int64
	MaxDepth int
}

// ReadJSONFile reads and parses a single JSON document from filePath.
// It enforces byte size limits, rejects trailing non-whitespace data, and preserves numeric literals without precision loss.
func ReadJSONFile(filePath string, opts Options) (any, error) {
	maxBytes := opts.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxInputBytes
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file %q: %w", filePath, err)
	}
	defer f.Close()

	// Read up to maxBytes + 1 to detect truncation
	limitedReader := io.LimitReader(f, maxBytes+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed reading file %q: %w", filePath, err)
	}

	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("file %q exceeds maximum allowed limit of %d bytes", filePath, maxBytes)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("file %q is empty", filePath)
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	var val any
	if err := dec.Decode(&val); err != nil {
		return nil, fmt.Errorf("invalid JSON in %q: %w", filePath, err)
	}

	// Ensure no unexpected trailing tokens exist
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("unexpected trailing data after top-level JSON in %q", filePath)
	}

	return val, nil
}
