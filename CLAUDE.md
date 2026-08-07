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

## Taskfile
Taskfile v3. `background: true` is not a supported task property in this version.
