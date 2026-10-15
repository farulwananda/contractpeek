package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farulwananda/contractpeek/internal/cli"
)

func TestCLI_EndToEndWorkflow(t *testing.T) {
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(origDir) }()

	var stdout, stderr bytes.Buffer

	// 1. Init
	code := cli.Run([]string{"init"}, &stdout, &stderr)
	if code != cli.ExitSuccess {
		t.Fatalf("init failed with code %d: %s", code, stderr.String())
	}

	// 2. Prepare sample JSON file
	sample1 := filepath.Join(tempDir, "sample-v1.json")
	_ = os.WriteFile(sample1, []byte(`{"data":{"user":{"id":"USR-1","name":"Alex","followers":100}}}`), 0600)

	// 3. Snapshot (new baseline)
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"snapshot", "--name", "user-api", "--file", sample1}, &stdout, &stderr)
	if code != cli.ExitSuccess {
		t.Fatalf("snapshot failed with code %d: %s", code, stderr.String())
	}

	// 4. List
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"list"}, &stdout, &stderr)
	if code != cli.ExitSuccess || !strings.Contains(stdout.String(), "user-api") {
		t.Fatalf("list unexpected: code %d, %s", code, stdout.String())
	}

	// 5. Check against identical response -> success (no drift)
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"check", "--name", "user-api", "--file", sample1}, &stdout, &stderr)
	if code != cli.ExitSuccess || !strings.Contains(stdout.String(), "matches approved baseline perfectly") {
		t.Fatalf("check identical failed: code %d, %s", code, stdout.String())
	}

	// 6. Check with breaking drift (followers changed from number to string)
	sample2 := filepath.Join(tempDir, "sample-v2-drift.json")
	_ = os.WriteFile(sample2, []byte(`{"data":{"user":{"id":"USR-1","name":"Alex","followers":"100"}}}`), 0600)

	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"check", "--name", "user-api", "--file", sample2, "--fail-on", "potential_breaking"}, &stdout, &stderr)
	if code != cli.ExitDrift {
		t.Fatalf("expected ExitDrift (1) for type change, got code %d. Output: %s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "type_changed") && !strings.Contains(stdout.String(), "POTENTIAL_BREAKING") {
		t.Fatalf("expected type drift output, got: %s", stdout.String())
	}

	// 7. Baseline update with --approve
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"baseline", "update", "--name", "user-api", "--file", sample2, "--approve"}, &stdout, &stderr)
	if code != cli.ExitSuccess {
		t.Fatalf("baseline update failed: code %d: %s", code, stderr.String())
	}

	// 8. Re-check sample2 -> now matches updated baseline
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"check", "--name", "user-api", "--file", sample2}, &stdout, &stderr)
	if code != cli.ExitSuccess {
		t.Fatalf("re-check after approve failed: code %d: %s", code, stdout.String())
	}
}

func TestCLI_MissingFlagsValidation(t *testing.T) {
	var stdout, stderr bytes.Buffer

	// Missing --name in check
	code := cli.Run([]string{"check"}, &stdout, &stderr)
	if code != cli.ExitUsage {
		t.Errorf("expected ExitUsage (2), got %d", code)
	}

	// Missing --file in snapshot
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"snapshot", "--name", "demo"}, &stdout, &stderr)
	if code != cli.ExitUsage {
		t.Errorf("expected ExitUsage (2), got %d", code)
	}
}
