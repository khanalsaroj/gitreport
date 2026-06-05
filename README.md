# gitreport

**gitreport** is a developer-focused CLI tool that transforms raw Git commit history into structured, high-quality engineering reports using AI.

It analyzes commits across one or multiple repositories, intelligently groups related changes (features, bug fixes, refactors), and generates concise, leadership-ready summaries. The tool eliminates the manual effort of writing weekly reports and provides clear visibility into engineering progress.

Designed for modern teams, gitreport supports flexible time filters, author-based breakdowns, and customizable AI prompts—making it suitable for both individual developers and team-level reporting.

---

## Features

- **Two report modes**: commit-message based (`summary`) or diff-based (`hard-summary`)
- **Multi-repo support**: analyze one repo, a list, or scan recursively
- **Author grouping**: break down contributions by engineer
- **Automated setup**: use `init` to quickly bootstrap your local configuration
- **Streaming output**: results print progressively, no waiting
- **Flexible formats**: `text`, `markdown`, `json`
- **Prompt-driven**: all AI prompts live in `gitreport.yaml` — no hardcoded strings

---

## ⚙️ Installation

### 🪟 Windows (PowerShell installer)

Open **PowerShell as Administrator**:

```powershell
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser
```

```powershell
iwr -useb https://raw.githubusercontent.com/khanalsaroj/gitreport/refs/heads/main/main/install.ps1 | iex
```

> ***Restart your terminal after installation.***

### Verify Installation

```bash
gitreport --help
```

### 🐧 Linux

> **Coming soon**

Or download a prebuilt binary for your platform from the [Releases](https://github.com/khanalsaroj/gitreport/releases)
page.

---

## Configuration

The easiest way to configure **gitreport** is to use the `init` command:

```bash
gitreport init
```

This will automatically create the required configuration files in your home directory:
- `~/.gitreport/setting.json` (API keys and model settings)
- `~/.gitreport/config/gitreport.yaml` (AI prompt templates)

---

### Manual Configuration (Optional)

If you prefer to configure the tool manually, create the following files in your home directory.

**Example (Windows):**
```bash
C:\Users\<user>\.gitreport\
```

### 1. Settings File

Create a file at `~/.gitreport/setting.json`:

```json
{
  "OPENAI_API_KEY": "sk-or-v1-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "OPENAI_BASE_URL": "https://openrouter.ai/api/v1/chat/completions",
  "OPENAI_MODEL": "nvidia/nemotron-3-super-120b-a12b:free"
}
```

Notes:
* `OPENAI_API_KEY` (required): Your API key
* `OPENAI_BASE_URL` (optional): Override for custom providers (Azure, Ollama, Groq, etc.)
* `OPENAI_MODEL` (optional): Model to use

---

### 2. Summary Configuration

Create the summary configuration file at `~/.gitreport/config/gitreport.yaml`:

> **Required:** This file must exist when using `summary` and `hard-summary` mode.

> **Tip:** `gitreport init` already writes this file for you from the
> configuration baked into the binary, so manual setup is rarely needed.

**Example (default template):**
[default.yaml](https://raw.githubusercontent.com/khanalsaroj/gitreport/refs/heads/main/internal/config/default.yaml)

---

### About `gitreport.yaml`

The `gitreport.yaml` file defines the prompt templates used by the AI model to generate reports.
You can fully customize these templates to match your reporting style, level of detail, and audience.

---

### Repository-Specific Configuration (Optional)

You can override the global configuration on a per-repository basis.

To do this, add a config file inside your repository:

```bash
config/gitreport.yaml
```

This allows different repositories to use tailored prompt templates while keeping a global default.

### Configuration Resolution Order

gitreport loads the first configuration it finds, in this order:

1. `$GITREPORT_CONFIG` — explicit path via environment variable
2. `./config/gitreport.yaml` — project-local config (repo-specific override)
3. `~/.gitreport/config/gitreport.yaml` — user-level config (written by `init`)
4. The default configuration embedded in the binary

Because the default is embedded, gitreport works out of the box even before
`init` has been run — only an API key is required.


---

## Usage

### Summary (commit messages)

```bash
# Last week of commits in current repo
gitreport summary --week 1

# Last 3 days, by author Name
gitreport summary --days 3 --author John Doe

# Last month, markdown output saved to file
gitreport summary --month 1 --format markdown --output report.md

# Multiple repos
gitreport summary --week 1 --projects /path/to/repo1,/path/to/repo2

```

### Hard Summary (code diffs)

```bash
# Deep analysis of last week's diffs
gitreport hard-summary --week 1

# Leadership report in markdown
gitreport hard-summary --week 1 --format markdown

# Specific repos, JSON output
gitreport hard-summary --days 5 \
  --projects /srv/api,/srv/frontend \
  --format json \
  --output weekly.json
```

---

## Flags

| Flag         | Type   | Description                                       |
|--------------|--------|---------------------------------------------------|
| `--week`     | int    | Look back N weeks (mutually exclusive)            |
| `--days`     | int    | Look back N days (mutually exclusive)             |
| `--month`    | int    | Look back N months (mutually exclusive)           |
| `--author`   | string | Commit by Author Name                             |
| `--projects` | string | Comma-separated list of repo paths                |
| `--format`   | string | Output format: `text`, `markdown`, `json`         |
| `--output`   | string | Write output to file instead of stdout            |

Run `gitreport --version` to print the build version.

> **Note:** Only one of `--week`, `--days`, `--month` may be used per invocation.

> **Note:** Additional output formats (for example `slack` and `html`) can be
> defined in `gitreport.yaml` and selected with `--format`; the model is
> instructed using that format's description.

---

## Development

Requires Go (see [`go.mod`](go.mod) for the minimum version).

```bash
# Build the binary
go build -o gitreport .

# Run the test suite
go test ./...

# Vet and format checks
go vet ./...
gofmt -l .
```

A `Makefile` wraps the common tasks:

```bash
make build   # compile ./gitreport
make test    # go test ./...
make check   # vet + gofmt + test
```

The default prompt configuration lives at
[`internal/config/default.yaml`](internal/config/default.yaml) and is embedded
into the binary at build time.

---
