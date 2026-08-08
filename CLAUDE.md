# Job Search Pipeline

## Key commands
- `task generate` - regenerate sqlc queries and templ templates (run after editing `.sql` or `.templ` files)
- `task dev` - runs `generate` first, then starts server on :8080
- `task test` / `task coverage` / `task lint` - all run `generate` as a dep automatically

## Generated files (gitignored)
`internal/db/db.go`, `internal/db/models.go`, `internal/db/queries.sql.go`, `internal/ui/*_templ.go`
These don't exist in the repo — `task generate` (or any task that deps on it) creates them.

## Dev loop
No live-reload. Edit → Ctrl-C → `task dev`. `air` is planned for a future version.

## API request schemas
`readJSON` rejects unknown fields, so a wrong field name is a 400 rather than a silent drop.
The field tables in `.claude/commands/job-search.md` ("API contract" section) are the schema the
agent running that command works from. Changing an API request struct means updating the matching
table — `TestCommandDocMatchesSchema` in `internal/api/contract_test.go` fails until you do.

## Taskfile
Taskfile v3. `background: true` is not a supported task property in this version.
