package report_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/farulwananda/contractpeek/internal/compare"
	"github.com/farulwananda/contractpeek/internal/report"
)

func TestPrintReport_TableAndMarkdown(t *testing.T) {
	res := &compare.Result{
		ContractName: "api-user",
		Summary: compare.Summary{
			Breaking: 1,
		},
		Changes: []compare.Change{
			{
				Path:        "#/id",
				Kind:        compare.KindRequiredPathMissing,
				Severity:    compare.SeverityBreaking,
				BeforeTypes: []string{"string"},
				AfterTypes:  []string{},
				Reason:      "Required field missing",
			},
		},
	}

	// 1. Table
	var tableBuf bytes.Buffer
	if err := report.PrintReport(res, "table", &tableBuf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tableBuf.String(), "CONTRACTPEEK AUDIT: api-user") {
		t.Errorf("expected table header, got: %s", tableBuf.String())
	}
	if !strings.Contains(tableBuf.String(), "[BREAKING]") {
		t.Errorf("expected [BREAKING], got: %s", tableBuf.String())
	}

	// 2. Markdown
	var mdBuf bytes.Buffer
	if err := report.PrintReport(res, "markdown", &mdBuf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mdBuf.String(), "## Contractpeek Report: `api-user`") {
		t.Errorf("expected markdown header, got: %s", mdBuf.String())
	}
	if !strings.Contains(mdBuf.String(), "| `BREAKING` | `#/id` |") {
		t.Errorf("expected markdown table row, got: %s", mdBuf.String())
	}

	// 3. JSON
	var jsonBuf bytes.Buffer
	if err := report.PrintReport(res, "json", &jsonBuf); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(jsonBuf.String()), "{") {
		t.Errorf("expected json object, got: %s", jsonBuf.String())
	}
}
