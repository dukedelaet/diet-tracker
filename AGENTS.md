# AGENTS.md

Diet Tracker: a Go CLI app for logging weight, meals, and exercise into a local SQLite database, with charts that render HTML (Chart.js) and open in the browser.

## Commands

```bash
make run                     # go run with DIET_DB_PATH=./app.db (pass ARGS=... to forward flags)
go install ./cmd/diet        # install binary (also: make install)
go test ./...                # tests so far are unit-level, no DB required
go run ./cmd/diet <args>     # run manually; DB path from DIET_DB_PATH or XDG data dir
sqlite3 app.db               # inspect local dev db (Makefile 'inspect' targets the macOS prod path)
scripts/seed.sh              # seed meal templates (reads the diet binary from PATH)
```

There is no CI config, linter config, or test harness beyond `go test`.

## Architecture

Two packages:

- `cmd/diet` — CLI only. One file per command group (`weight-handlers.go`, `meal-handlers.go`, `exercise-handlers.go`, `chart-handlers.go`), all in `package main`.
- `internal/db` — sqlc-generated data access (marked "DO NOT EDIT" in `.go` files).
- `migrations/` — goose SQL migrations, embedded via `//go:embed` in `migrations/migrations.go`.

Control flow: `main.go` opens SQLite (modernc.org/sqlite, pure Go — no cgo), applies goose migrations from the embedded FS on **every startup**, then wires up a `daved/clic` command tree and dispatches `cmd.Handle(ctx)`. Each command is a small struct holding an `io.Writer` and `*db.Queries`, either implementing `HandleCommand(ctx)` directly (group parent commands) or exposing a method returning `clic.HandlerFunc` (leaf commands, e.g. `LogMeal().LogMeal()`).

Data flow: handler → `db.Queries` (sqlc) → SQLite. Dates are stored as `TEXT` (`time.DateOnly`, i.e. `YYYY-MM-DD`). Calorie totals are computed in Go (`CalculateCals`), not in SQL — except `ListDailyCaloriesFromDate` which sums stored `meal_logs.calories`.

## Gotchas / conventions

- **Adding a query**: write/modify the SQL in `internal/db/*.sql`, then regenerate with `sqlc generate` (config in `sqlc.yaml`, schema = `migrations/`). sqlc v1.30.0 is the pinned version in generated file headers. `sqlc` is not on this machine's PATH. Never hand-edit `*.sql.go`, `models.go`, or `db.go`.
- **Schema changes**: add a new numbered goose file in `migrations/` (existing files are `000001_*`, `00002_*`, `00003_*` — the zero-padding is inconsistent; keep the numeric ordering correct). Migrations run at startup, not via any make target.
- **DB location**: single source of truth — `DIET_DB_PATH` > `XDG_DATA_HOME/diet-tracker/app.db` > OS default (macOS Application Support / Linux `.local/share`). CLI and TUI share the same path by default; `make run` no longer overrides it.
- **Chart start dates**: CLI flag `-s` takes `MM-DD-YYYY` (`inputDateFormat = "01-02-2006"` in chart-handlers.go), but all stored dates are `YYYY-MM-DD`. `parseStartDate` converts. Don't mix these formats.
- **Charts**: rendered to `os.TempDir()` as a timestamped HTML file using `text/template` + `//go:embed templates/*` (base.html + one JS template per chart type), then opened with `open`/`xdg-open`/`cmd /c start`. New chart types need: a template file, a parsed template var at top of chart-handlers.go, a data type, a handler, and wiring in main.go.
- **Error handling convention**: leaf handlers return errors; main.go special-cases `sql.ErrNoRows` → prints "No entries in db" instead of crashing. Sentinel errors (`noMacrosErr`, `missingNameErr`, etc.) are package-level vars checked with `errors.Is`.
- **Testing pattern** (see meal-handlers_test.go): table-driven, pure-logic functions only; handler structs accept nil `io.Writer` and `&db.Queries{}` because only validation logic is exercised. No in-memory-DB test setup exists.
- **Portion logging**: `log meal -p 50` stores *computed* macro values on the log row (meal_logs carries its own protein/carbs/fat/calories since migration 00003); a full portion goes through a different sqlc query. Keep both paths in sync when changing macros.
- **Weight input is in pounds** throughout; nothing is converted to kg anywhere.
