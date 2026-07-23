# Job Search Pipeline

A self-hosted job search tracker and AI assistant built for senior engineers running a focused, high-signal search. You keep your jobs, notes, and generated documents in one place — a local Kanban board that lives in a SQLite file you own — and Claude Code does the heavy lifting: evaluating fit, writing tailored resumes and cover letters, and generating interview prep.

The workflow is:
1. Run `/job-search init` in Claude Code to save your profile (resume, salary range, preferences).
2. Run `/job-search add <url>` on any job posting. Claude fetches the JD, scores fit 1–10 against your profile, writes a summary, and adds the job to your board.
3. Open `http://localhost:8080` to see your Kanban board. Drag jobs between stages as you progress. Click any card to see the full evaluation, activity log, and generated documents.
4. Use `/job-search resume`, `/job-search cover-letter`, and `/job-search prep` to generate tailored documents for jobs you want to pursue. The board shows a stale indicator (⚠) when your profile has changed since a document was generated, so you know what to regenerate.
5. Use `/job-search eval` to re-score a job when new information comes in (salary disclosed, role clarified, etc.).

**Tech stack:** Go, chi, templ, HTMX, Sortable.js, SQLite, sqlc. No external services, no telemetry, no accounts.

---

## Quick start

### Option 1 — Docker (no Go required)

```bash
curl -O https://raw.githubusercontent.com/sdroscher/job-search-pipeline/main/docker-compose.yml
docker compose up
```

The image is pulled from GHCR automatically. Data and generated files persist in `./data` and `./output`.

### Option 2 — Pre-built binary (no Go required)

Download the archive for your platform from the [latest release](https://github.com/sdroscher/job-search-pipeline/releases/latest), then:

```bash
tar -xzf job-search-pipeline_darwin_arm64.tar.gz   # adjust for your platform
cd job-search-pipeline_darwin_arm64
./job-search-pipeline
```

Keep `static/` in the same directory as the binary — it holds the CSS and JS the web UI needs.

### Claude Code skill (both options)

The archive and Docker setup both include `.claude/commands/job-search.md`. Copy it into your project (or `~/.claude/commands/`) so Claude Code picks up the `/job-search` commands:

```bash
mkdir -p ~/.claude/commands
cp .claude/commands/job-search.md ~/.claude/commands/
```

Open Claude Code in the directory where your `data/` and `output/` folders live and run `/job-search init` to set up your profile.

---

### Build from source

Requires Go 1.25+, Task, templ, sqlc, golangci-lint, air.

| Tool | Install |
|---|---|
| [Task](https://taskfile.dev) | `brew install go-task` |
| [templ](https://templ.guide) | `go install github.com/a-h/templ/cmd/templ@latest` |
| [sqlc](https://sqlc.dev) | `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest` |
| [golangci-lint](https://golangci-lint.run) | `brew install golangci-lint` |
| [air](https://github.com/air-verse/air) | `go install github.com/air-verse/air@latest` |

```bash
git clone https://github.com/sdroscher/job-search-pipeline
cd job-search-pipeline
task dev
```

`task dev` uses [air](https://github.com/air-verse/air) for live-reload — the server restarts automatically when you change `.go`, `.templ`, or `.sql` files.

Open `http://localhost:8080` — you'll see an empty board with seven stages: Evaluated → Applied → AI Assessment → Screening → Interviewing → Final Round → Offer.

Open Claude Code in this directory and run `/job-search init` to save your profile.

---

## The board

`http://localhost:8080` is a Kanban board with one column per pipeline stage. Each card shows the company, role, fit score, and verdict (green/yellow/red). Cards are draggable between columns — dropping a card on a new column updates the stage immediately via HTMX. Clicking a card opens a detail panel with the full AI evaluation, salary, remote status, positives, concerns, and activity log.

A ⚠ icon on a card means one or more generated documents (resume, cover letter) were created against an older version of your profile. Run `/job-search resume <id>` or `/job-search cover-letter <id>` to regenerate.

The detail panel is resizable — drag the left edge (the thin vertical gutter between the board and panel) to adjust its width. The width persists across page reloads.

---

## Claude Code skill

Run these commands from any Claude Code session open in this directory. The skill reads `$JOB_PIPELINE_URL` (default `http://localhost:8080`) so the server must be running.

### `/job-search init`

First-time setup. Claude asks for your resume (paste or file path), salary range, remote preference, location, preferred industries, tech stack, green flags, red flags, and writing voice notes. Saves everything to `/api/profile`.

### `/job-search add <url>`

Paste a job URL from Greenhouse, Ashby, Lever, BambooHR, SmartRecruiters, or any careers page. Claude:
- Fetches and parses the job description (native API for Greenhouse/Ashby/Lever/SmartRecruiters, HTML scraper for BambooHR and generic pages)
- Detects if the URL is from a job aggregator (LinkedIn, Indeed, Glassdoor, Jobgether, Jobright) and automatically searches for the company's direct ATS posting — offering to use that instead if found
- Fetches the company's careers/about page to extract named values and mission
- Scores fit 1–10 across salary alignment (25%), remote/location match (20%), tech stack (20%), green/red flags (20%), and role seniority (15%)
- Sets verdict: green (8–10), yellow (5–7), red (1–4)
- Writes 3–5 specific positives and 3–5 specific concerns
- Generates a one-paragraph summary calibrated to your profile
- Asks in one turn: any missing salary/location info, additional context (recruiter reach-out, referral, etc.), and whether you have a networking contact at the company
- Assigns a short memorable slug ID (e.g. `stripe-staff-swe`) you can use in future commands
- Adds the job to your board in the Evaluated stage
- If the URL can't be parsed, prompts you to paste the JD directly

### `/job-search resume <job-id>`

Generates a tailored resume in Markdown. Claude reorders your experience bullets so the most JD-relevant ones lead in each role, adjusts your summary to mirror the JD's language, and surfaces the specific technologies the role calls for — without inventing experience. Saves to `$OUTPUT_DIR/resume-<company>-<role>.md` and registers the document as an artifact (so the board can track freshness).

### `/job-search cover-letter <job-id>`

Generates a four-paragraph cover letter in your voice:
- **P1:** What drew you to this company specifically — named values, specific product, remote-first culture, etc.
- **P2:** One achievement story that connects your background to the team's actual work.
- **P3:** Three numbered achievements most relevant to the JD, specific and measurable where possible.
- **P4:** Availability and closing invitation.

Uses your `writing_voice_md` guide and `cover_letter_sample` if you provided them during init. Saves to `$OUTPUT_DIR/cover-letter-<company>-<role>.md`.

### `/job-search prep <job-id>`

Generates an interview prep document with 5–7 behavioral questions (each with a suggested STAR story from your experience), 5–7 technical questions based on the role's stack, a research checklist (company news, blog posts, team context), and 5–7 questions to ask them about team structure, on-call, tech debt, and growth. Saves to `$OUTPUT_DIR/prep-<company>-<role>.md`.

### `/job-search eval <job-id>`

Re-runs the fit evaluation with the same scoring rubric as `add`. Use this when salary is disclosed, the role scope is clarified, or your profile has changed. Updates fit score, verdict, positives, and concerns in the DB and logs the change to the activity log (e.g. "fitScore 6 → 9").

### `/job-search reach-out <job-id>`

Drafts a personalized LinkedIn message or email to a networking contact at the company. Asks how you know them, what you want from the outreach, and which channel. Produces a concise draft (LinkedIn: under 150 words, email: under 250) grounded in your shared context — never "I saw a job posting."

### `/job-search compare <job-id-1> <job-id-2>`

Side-by-side comparison of two jobs to help decide which to prioritise next. Fetches both jobs and your profile, builds a comparison table (role, fit score, verdict, salary, remote, stage), highlights up to three key positives and concerns per role, and writes a 3–5 sentence recommendation naming which to pursue first and why.

---

## The detail panel

Clicking any card on the board opens a detail panel on the right showing the full AI evaluation (fit score, verdict, positives, concerns), salary, remote status, company values, activity log, and any generated artifacts (resume, cover letter, prep doc). Artifacts are lazy-loaded on expand and rendered as formatted Markdown.

The panel displays the job's slug ID below the posting link, with a copy-to-clipboard button for quick reference in commands.

---

## Closing jobs

The detail panel for any active job shows four **Close as…** buttons: Rejected, Listing Withdrawn, Declined, and Won't Apply. Clicking one moves the job off the active board immediately. A collapsible **Closed** section below the kanban columns holds all closed jobs — click any closed card to reopen the detail panel, where a **Re-open → Evaluated** button returns the job to the active board.

---

## Profile

`http://localhost:8080/profile` is a form where you can view and edit your profile directly in the browser. Changes mark all existing artifacts as stale (⚠) so you know which documents need regenerating.

---

## Config

| Flag | Env | Default | Description |
|---|---|---|---|
| `--port` | `PORT` | `8080` | HTTP listen port |
| `--database-url` | `DATABASE_URL` | `./data/pipeline.db` | SQLite file path |
| `--output-dir` | `OUTPUT_DIR` | `./output` | Resume/cover letter output directory |

```bash
./bin/job-search-pipeline --help
```

---

## Development

```bash
task dev         # start server with air live-reload (restarts on .go/.templ/.sql changes)
task generate    # regenerate sqlc queries and templ templates
task test        # run all tests (unit + feature)
task lint        # golangci-lint
task coverage    # test coverage summary (hand-written code only)
task build       # compile to bin/job-search-pipeline
```

Generated files (`internal/db/db.go`, `internal/db/models.go`, `internal/db/queries.sql.go`, `internal/ui/*_templ.go`) are excluded from git and regenerated on each build. Edit `sql/queries.sql` or `internal/ui/*.templ` and run `task generate` to update them.

### Supported ATS platforms

| Platform | Method |
|---|---|
| Greenhouse | JSON API |
| Ashby | JSON API |
| Lever | JSON API |
| SmartRecruiters | JSON API |
| BambooHR | HTML scraper |
| Jobgether | HTML scraper |
| Jobright | HTML scraper |
| LinkedIn | HTML scraper (detects aggregator, prompts for direct link) |
| Indeed | HTML scraper (detects aggregator, prompts for direct link) |
| Glassdoor | HTML scraper (detects aggregator, prompts for direct link) |
| Generic careers pages | HTML scraper |
