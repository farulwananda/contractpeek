package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/farulwananda/contractpeek/internal/baseline"
	"github.com/farulwananda/contractpeek/internal/compare"
	"github.com/farulwananda/contractpeek/internal/input"
	"github.com/farulwananda/contractpeek/internal/report"
	"github.com/farulwananda/contractpeek/internal/shape"
)

// Exit codes following Section 14 of CONTRACTPEEK_PLAN.md
const (
	ExitSuccess = 0 // No findings meeting configured failure threshold
	ExitDrift   = 1 // Contract drift reached failure threshold
	ExitUsage   = 2 // Usage / config / I/O / invalid JSON error
)

const rootHelp = `contractpeek - API Contract & Schema Drift Inspector

Usage:
  contractpeek <command> [flags]

Commands:
  init              Create minimal local project config and baseline directory (.contractpeek/)
  snapshot          Infer observed contract and save a NEW named baseline from JSON sample(s)
  check             Inspect response sample(s) against an approved baseline (read-only)
  baseline update   Explicitly replace an approved baseline from reviewed sample file(s)
  list              Show all local approved baseline contracts
  diff              Directly compare two baseline/snapshot JSON files
  version           Show contractpeek version information

Flags:
  -h, --help        Show help for contractpeek or a subcommand
`

// arrayFlags supports repeated flags like --file sample1.json --file sample2.json
type arrayFlags []string

func (a *arrayFlags) String() string {
	return strings.Join(*a, ", ")
}

func (a *arrayFlags) Set(val string) error {
	*a = append(*a, val)
	return nil
}

// Run executes the CLI application and returns an exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, rootHelp)
		return ExitSuccess
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "help", "-h", "--help":
		fmt.Fprint(stdout, rootHelp)
		return ExitSuccess

	case "version", "-v", "--version":
		fmt.Fprintln(stdout, "contractpeek v0.1.0")
		return ExitSuccess

	case "init":
		return runInit(cmdArgs, stdout, stderr)

	case "snapshot":
		return runSnapshot(cmdArgs, stdout, stderr)

	case "check":
		return runCheck(cmdArgs, stdout, stderr)

	case "baseline":
		return runBaseline(cmdArgs, stdout, stderr)

	case "list":
		return runList(cmdArgs, stdout, stderr)

	case "diff":
		return runDiff(cmdArgs, stdout, stderr)

	default:
		fmt.Fprintf(stderr, "contractpeek: unknown command %q\nRun 'contractpeek --help' for usage.\n", cmd)
		return ExitUsage
	}
}

func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitSuccess
		}
		return ExitUsage
	}

	store, err := baseline.NewStore(".")
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek init: %v\n", err)
		return ExitUsage
	}

	if err := store.Init(); err != nil {
		fmt.Fprintf(stderr, "contractpeek init: %v\n", err)
		return ExitUsage
	}

	fmt.Fprintln(stdout, "contractpeek: initialized .contractpeek/ directory and config.json")
	return ExitSuccess
}

func runSnapshot(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	fs.SetOutput(stderr)

	name := fs.String("name", "", "Contract baseline name (e.g. 'tikhub-profile')")
	var files arrayFlags
	fs.Var(&files, "file", "Path to JSON sample file (can be repeated)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitSuccess
		}
		return ExitUsage
	}

	if *name == "" {
		fmt.Fprintln(stderr, "contractpeek snapshot: --name flag is required")
		return ExitUsage
	}
	if len(files) == 0 {
		fmt.Fprintln(stderr, "contractpeek snapshot: at least one --file flag is required")
		return ExitUsage
	}

	store, err := baseline.NewStore(".")
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek snapshot: %v\n", err)
		return ExitUsage
	}

	cfg, err := store.LoadConfig()
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek snapshot: %v\n", err)
		return ExitUsage
	}
	samples, err := loadSamples(files, cfg.Defaults.MaxInputBytes)
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek snapshot: %v\n", err)
		return ExitUsage
	}

	contract, err := shape.InferFromSamples(samples, shape.InferOptions{MaxDepth: cfg.Defaults.MaxDepth})
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek snapshot: inference failed: %v\n", err)
		return ExitUsage
	}

	if err := store.SaveBaseline(*name, contract, false); err != nil {
		fmt.Fprintf(stderr, "contractpeek snapshot: %v\n", err)
		return ExitUsage
	}

	fmt.Fprintf(stdout, "contractpeek: saved new baseline %q (%d fields inferred from %d sample(s))\n",
		*name, len(contract.Fields), len(samples))
	return ExitSuccess
}

func runCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)

	name := fs.String("name", "", "Contract baseline name to verify against")
	var files arrayFlags
	fs.Var(&files, "file", "Path to JSON response sample file (can be repeated)")
	format := fs.String("format", "table", "Output format: 'table', 'json', or 'markdown'")
	failOn := fs.String("fail-on", "", "Failure exit code threshold: 'breaking', 'potential_breaking', or 'none'")
	outputFile := fs.String("output", "", "Optional file destination for report")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitSuccess
		}
		return ExitUsage
	}

	if *name == "" {
		fmt.Fprintln(stderr, "contractpeek check: --name flag is required")
		return ExitUsage
	}
	if len(files) == 0 {
		fmt.Fprintln(stderr, "contractpeek check: at least one --file flag is required")
		return ExitUsage
	}

	store, err := baseline.NewStore(".")
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek check: %v\n", err)
		return ExitUsage
	}

	cfg, err := store.LoadConfig()
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek check: %v\n", err)
		return ExitUsage
	}
	base, err := store.LoadBaseline(*name)
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek check: failed loading baseline %q: %v\n", *name, err)
		return ExitUsage
	}

	samples, err := loadSamples(files, cfg.Defaults.MaxInputBytes)
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek check: %v\n", err)
		return ExitUsage
	}

	current, err := shape.InferFromSamples(samples, shape.InferOptions{MaxDepth: cfg.Defaults.MaxDepth})
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek check: %v\n", err)
		return ExitUsage
	}

	// Read rule overrides from config
	var reqPaths, ignPaths []string
	if cRule, ok := cfg.Contracts[*name]; ok {
		reqPaths = cRule.RequiredPaths
		ignPaths = cRule.IgnoredPaths
	}

	result := compare.Compare(*name, base, current, compare.Options{
		RequiredPaths: reqPaths,
		IgnoredPaths:  ignPaths,
	})

	var out io.Writer = stdout
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(stderr, "contractpeek check: failed opening output file: %v\n", err)
			return ExitUsage
		}
		defer f.Close()
		out = f
	}

	if err := report.PrintReport(result, *format, out); err != nil {
		fmt.Fprintf(stderr, "contractpeek check: failed rendering report: %v\n", err)
		return ExitUsage
	}

	effectiveFailOn := *failOn
	if effectiveFailOn == "" {
		effectiveFailOn = cfg.Defaults.FailOn
	}

	if shouldFailCheck(result, effectiveFailOn) {
		return ExitDrift
	}

	return ExitSuccess
}

func runBaseline(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "contractpeek baseline: subcommand required (e.g. 'baseline update')")
		return ExitUsage
	}

	subCmd := args[0]
	subArgs := args[1:]

	if subCmd != "update" {
		fmt.Fprintf(stderr, "contractpeek baseline: unknown subcommand %q\n", subCmd)
		return ExitUsage
	}

	fs := flag.NewFlagSet("baseline update", flag.ContinueOnError)
	fs.SetOutput(stderr)

	name := fs.String("name", "", "Contract baseline name to update")
	var files arrayFlags
	fs.Var(&files, "file", "Path to JSON sample file (can be repeated)")
	approve := fs.Bool("approve", false, "Explicitly confirm baseline replacement")

	if err := fs.Parse(subArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitSuccess
		}
		return ExitUsage
	}

	if *name == "" {
		fmt.Fprintln(stderr, "contractpeek baseline update: --name flag is required")
		return ExitUsage
	}
	if len(files) == 0 {
		fmt.Fprintln(stderr, "contractpeek baseline update: at least one --file flag is required")
		return ExitUsage
	}
	if !*approve {
		fmt.Fprintln(stderr, "contractpeek baseline update: --approve flag is required to confirm updating baseline contract")
		return ExitUsage
	}

	store, err := baseline.NewStore(".")
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek baseline update: %v\n", err)
		return ExitUsage
	}

	cfg, err := store.LoadConfig()
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek baseline update: %v\n", err)
		return ExitUsage
	}
	samples, err := loadSamples(files, cfg.Defaults.MaxInputBytes)
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek baseline update: %v\n", err)
		return ExitUsage
	}

	contract, err := shape.InferFromSamples(samples, shape.InferOptions{MaxDepth: cfg.Defaults.MaxDepth})
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek baseline update: %v\n", err)
		return ExitUsage
	}

	// Overwrite approved baseline atomically
	if err := store.SaveBaseline(*name, contract, true); err != nil {
		fmt.Fprintf(stderr, "contractpeek baseline update: %v\n", err)
		return ExitUsage
	}

	fmt.Fprintf(stdout, "contractpeek: successfully updated approved baseline %q (%d fields from %d sample(s))\n",
		*name, len(contract.Fields), len(samples))
	return ExitSuccess
}

func runList(args []string, stdout, stderr io.Writer) int {
	store, err := baseline.NewStore(".")
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek list: %v\n", err)
		return ExitUsage
	}

	names, err := store.List()
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek list: %v\n", err)
		return ExitUsage
	}

	if len(names) == 0 {
		fmt.Fprintln(stdout, "No approved contract baselines found in .contractpeek/baselines/")
		return ExitSuccess
	}

	fmt.Fprintf(stdout, "APPROVED CONTRACT BASELINES (%d):\n", len(names))
	for _, n := range names {
		b, err := store.LoadBaseline(n)
		if err == nil {
			fmt.Fprintf(stdout, "  - %-24s (%d fields, %d sample(s), updated %s)\n",
				b.Name, len(b.Fields), b.SampleCount, b.CreatedAt.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Fprintf(stdout, "  - %s\n", n)
		}
	}
	return ExitSuccess
}

func runDiff(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(stderr)

	oldPath := fs.String("old", "", "Path to older baseline JSON file (required)")
	newPath := fs.String("new", "", "Path to newer baseline JSON file (required)")
	format := fs.String("format", "table", "Output format: 'table', 'json', or 'markdown'")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitSuccess
		}
		return ExitUsage
	}

	if *oldPath == "" || *newPath == "" {
		fmt.Fprintln(stderr, "contractpeek diff: both --old and --new flags are required")
		return ExitUsage
	}

	oldBase, err := baseline.LoadBaselineFile(*oldPath)
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek diff: %v\n", err)
		return ExitUsage
	}

	newBase, err := baseline.LoadBaselineFile(*newPath)
	if err != nil {
		fmt.Fprintf(stderr, "contractpeek diff: %v\n", err)
		return ExitUsage
	}

	// Convert newBase fields to shape.InferredContract for comparison
	fieldMap := make(map[string]*shape.FieldInfo)
	for _, f := range newBase.Fields {
		fieldMap[f.Path] = &shape.FieldInfo{
			Path:              f.Path,
			Types:             f.Types,
			ObservedInSamples: f.ObservedInSamples,
			NullInSamples:     f.NullInSamples,
		}
	}
	currentContract := &shape.InferredContract{
		SampleCount: newBase.SampleCount,
		Fields:      fieldMap,
	}

	result := compare.Compare(fmt.Sprintf("%s vs %s", oldBase.Name, newBase.Name), oldBase, currentContract, compare.Options{})
	_ = report.PrintReport(result, *format, stdout)
	return ExitSuccess
}

func loadSamples(paths []string, maxBytes int64) ([]any, error) {
	var samples []any
	for _, p := range paths {
		v, err := input.ReadJSONFile(p, input.Options{MaxBytes: maxBytes})
		if err != nil {
			return nil, err
		}
		samples = append(samples, v)
	}
	return samples, nil
}

func shouldFailCheck(res *compare.Result, failOn string) bool {
	switch strings.ToLower(strings.TrimSpace(failOn)) {
	case "none":
		return false
	case "potential_breaking":
		return res.Summary.Breaking > 0 || res.Summary.PotentialBreaking > 0
	case "breaking", "":
		return res.Summary.Breaking > 0
	default:
		return res.Summary.Breaking > 0
	}
}

func isValidFailOn(failOn string) bool {
	switch strings.ToLower(strings.TrimSpace(failOn)) {
	case "none", "potential_breaking", "breaking", "":
		return true
	default:
		return false
	}
}

func isValidFormat(format string) bool {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "table", "json", "markdown":
		return true
	default:
		return false
	}
}
