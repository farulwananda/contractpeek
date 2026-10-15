package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/farulwananda/contractpeek/internal/compare"
)

// PrintReport renders comparison outcomes in table, json, or markdown format.
func PrintReport(res *compare.Result, format string, out io.Writer) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(res)

	case "markdown":
		return printMarkdown(res, out)

	default:
		return printTable(res, out)
	}
}

func printTable(res *compare.Result, out io.Writer) error {
	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "CONTRACTPEEK AUDIT: %s\n", res.ContractName)
	fmt.Fprintf(w, "Summary:\t%d breaking, %d potential breaking, %d info\n\n",
		res.Summary.Breaking, res.Summary.PotentialBreaking, res.Summary.Info)

	if len(res.Changes) == 0 {
		fmt.Fprintf(w, "No contract drift detected. Response matches approved baseline perfectly!\n")
		return w.Flush()
	}

	fmt.Fprintf(w, "SEVERITY\tPATH\tBEFORE\tAFTER\tREASON\n")
	for _, c := range res.Changes {
		beforeStr := strings.Join(c.BeforeTypes, "|")
		if beforeStr == "" {
			beforeStr = "-"
		}
		afterStr := strings.Join(c.AfterTypes, "|")
		if afterStr == "" {
			afterStr = "-"
		}

		fmt.Fprintf(w, "[%s]\t%s\t%s\t%s\t%s\n",
			strings.ToUpper(string(c.Severity)),
			c.Path,
			beforeStr,
			afterStr,
			c.Reason,
		)
	}
	return w.Flush()
}

func printMarkdown(res *compare.Result, out io.Writer) error {
	fmt.Fprintf(out, "## Contractpeek Report: `%s`\n\n", res.ContractName)
	fmt.Fprintf(out, "**Summary:** `%d breaking`, `%d potential breaking`, `%d info`\n\n",
		res.Summary.Breaking, res.Summary.PotentialBreaking, res.Summary.Info)

	if len(res.Changes) == 0 {
		fmt.Fprintf(out, "No contract drift detected. Response matches approved baseline perfectly!\n")
		return nil
	}

	fmt.Fprintf(out, "| Severity | Path | Previous Types | Current Types | Reason |\n")
	fmt.Fprintf(out, "|---|---|---|---|---|\n")
	for _, c := range res.Changes {
		beforeStr := strings.Join(c.BeforeTypes, " \\| ")
		if beforeStr == "" {
			beforeStr = "-"
		}
		afterStr := strings.Join(c.AfterTypes, " \\| ")
		if afterStr == "" {
			afterStr = "-"
		}

		fmt.Fprintf(out, "| `%s` | `%s` | %s | %s | %s |\n",
			strings.ToUpper(string(c.Severity)),
			c.Path,
			beforeStr,
			afterStr,
			c.Reason,
		)
	}
	return nil
}
