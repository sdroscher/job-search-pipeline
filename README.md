# Job Search Pipeline

Self-hosted AI-powered job tracking board with Claude Code integration.

## Quick start

### Requirements
- Go 1.25+
- [Task](https://taskfile.dev) (`brew install go-task`)
- [templ](https://templ.guide) (`go install github.com/a-h/templ/cmd/templ@latest`)
- [sqlc](https://sqlc.dev) (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`)
- [golangci-lint](https://golangci-lint.run) (`brew install golangci-lint`)
- Claude Code

### Setup

```bash
git clone https://github.com/you/job-search-pipeline
cd job-search-pipeline
cp .env.example .env
task dev
```

Open Claude Code in this directory, then run `/job-search init`.

### Config (via env or CLI flags)

| Flag | Env | Default | Description |
|---|---|---|---|
| `--port` | `PORT` | `8080` | HTTP port |
| `--database-url` | `DATABASE_URL` | `./data/pipeline.db` | SQLite path |
| `--output-dir` | `OUTPUT_DIR` | `./output` | Resume/cover letter output |

`./bin/job-search-pipeline --help` for full usage.

### Claude Code commands

| Command | Description |
|---|---|
| `/job-search init` | First-time profile setup |
| `/job-search add <url>` | Parse, evaluate, and add a job |
| `/job-search resume <job-id>` | Generate tailored resume |
| `/job-search cover-letter <job-id>` | Generate cover letter |
| `/job-search prep <job-id>` | Interview prep notes |
| `/job-search eval <job-id>` | Re-evaluate fit |

### Linting

Place your `.golangci.yml` at the repo root, then:

```bash
task lint
```

### Docker

```bash
docker compose up
```
