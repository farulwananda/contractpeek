# contractpeek

> **API Contract & Schema Drift Inspector**  
> *Detect breaking API changes before they break your integrations.*

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://img.shields.io/badge/CI-Passing-brightgreen?style=flat&logo=github-actions)](https://github.com/farulwananda/contractpeek/actions)

`contractpeek` is a fast, lightweight, dependency-free developer tool written in Go to track the observed schema of JSON API responses, compare new observations against approved local baselines, and detect contract drift and breaking changes in your CI pipelines.

Designed for backend developers and data engineers integrating third-party APIs, webhooks, and scrapers (Apify, TikHub, social data APIs, payment gateways) where responses frequently evolve, add optional fields, or change data types without notice.

---

## Key Principles

- **Zero Secret & Value Persistence:** Infers compact pointer paths and types (`#/data/user/id: string`). Raw response values and payloads are **never persisted** to baseline files or Git.
- **Git-Friendly Local Baselines:** Baselines live in `.contractpeek/baselines/` inside your repository as human-readable, deterministic JSON.
- **Explicit Review Workflow:** `contractpeek check` is **strictly read-only**. Baselines are only updated when explicitly approved via `contractpeek baseline update --approve`.
- **Epistemically Honest Drift Detection:** Accurately distinguishes between `field_not_observed`, `type_changed`, `required_path_missing`, and unobservable fields in empty arrays (`[]`).
- **Dependency-Free:** Built exclusively using the Go standard library.

---

## Installation

### Using Go

```bash
go install github.com/farulwananda/contractpeek/cmd/contractpeek@latest
```

### From Source

```bash
git clone https://github.com/farulwananda/contractpeek.git
cd contractpeek
go build -o contractpeek ./cmd/contractpeek
```

---

## Quick Start

### 1. Initialize Local Project

In your backend repository, initialize the local `.contractpeek/` directory and configuration:

```bash
contractpeek init
```

This creates `.contractpeek/config.json` and the `.contractpeek/baselines/` directory.

### 2. Infer and Save Initial Baseline Snapshot

Save an initial observed baseline contract from one or more sample JSON responses:

```bash
contractpeek snapshot --name user-profile --file testdata/original.json
```

Or provide multiple sample files to combine observations into a unified type union:

```bash
contractpeek snapshot --name user-profile \
  --file samples/user-1.json \
  --file samples/user-2.json
```

### 3. Check for Breaking Changes (CI & Local)

Inspect a fresh response against your approved baseline (read-only):

```bash
contractpeek check --name user-profile --file testdata/changed-types.json
```

Output:

```text
CONTRACTPEEK AUDIT: user-profile
Summary:	0 breaking, 2 potential breaking, 0 info

SEVERITY               PATH                                  BEFORE   AFTER    REASON
[POTENTIAL_BREAKING]   #/data/posts/*/likes                  number   string   Observed types differ from approved baseline
[POTENTIAL_BREAKING]   #/data/profile/followers              number   string   Observed types differ from approved baseline
```

Format as a pull-request Markdown summary:

```bash
contractpeek check --name user-profile \
  --file testdata/changed-types.json \
  --format markdown \
  --output contract-report.md
```

### 4. Review and Update Approved Baseline

When an API change is reviewed and intentionally accepted, replace the baseline with explicit approval:

```bash
contractpeek baseline update --name user-profile --file testdata/changed-types.json --approve
```

Commit the resulting `.contractpeek/baselines/user-profile.json` in Git alongside your code mapper updates.

### 5. Compare Two Baselines Directly

```bash
contractpeek diff --old .contractpeek/baselines/v1.json --new .contractpeek/baselines/v2.json
```

---

## CLI Reference

| Command | Flags | Description |
|---|---|---|
| `contractpeek init` | | Initialize `.contractpeek/` and default `config.json` |
| `contractpeek snapshot` | `--name <str>`, `--file <path>` (repeatable) | Infer and save a new approved contract baseline |
| `contractpeek check` | `--name <str>`, `--file <path>`, `--format table\|json\|markdown`, `--fail-on <level>`, `--output <file>` | Inspect sample responses against approved baseline (read-only) |
| `contractpeek baseline update` | `--name <str>`, `--file <path>`, `--approve` | Explicitly update approved baseline from reviewed samples |
| `contractpeek list` | | List all approved baselines, field counts, and sample counts |
| `contractpeek diff` | `--old <file>`, `--new <file>`, `--format table\|json\|markdown` | Directly compare two baseline documents |
| `contractpeek version` | | Show tool version |

### Exit Codes

| Code | Meaning |
|---|---|
| `0` | Success: no findings meeting `--fail-on` threshold |
| `1` | Drift detected: findings reached `--fail-on` severity |
| `2` | Usage error, invalid JSON, or unreadable file path |

---

## Testing & Verification

```bash
# Run unit and race detection tests
go test -v ./...
go test -race ./...
```

Verify drift detection using the included mock test fixtures:

```bash
# Expected: exit 1 (detects type changes in followers and likes)
go run ./cmd/contractpeek check --name user-profile --file testdata/changed-types.json --fail-on potential_breaking
```

---

## License

This project is licensed under the [MIT License](LICENSE).
