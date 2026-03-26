# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

DSA (Database Session Activator) is a Go CLI tool that prevents a Gabia-hosted PostgreSQL from going idle. It runs in two modes, invoked externally by a scheduler (cron or systemd timer):

- **`keepalive`** (default): Runs 3 queries against the DB in order. Sends a KakaoWork failure alert only if all 3 fail.
- **`daily-report`**: Reads the previous day's JSONL log, aggregates stats, and sends a KakaoWork summary.

## Commands

```bash
# Run (keepalive mode)
go run ./cmd/app

# Run (daily-report mode)
RUN_MODE=daily-report go run ./cmd/app

# Build binary
go build -o dsa ./cmd/app        # Linux
go build -o dsa.exe ./cmd/app    # Windows

# Run tests
go test ./...

# Run a single package's tests
go test ./internal/report/...
```

## Architecture

Entry point: `cmd/app/main.go` → loads config → initializes `logx.Logger` → calls `job.Run`.

```
internal/
  config/     Environment variable loading and validation (Config struct)
  job/        Top-level orchestration: runKeepalive / runDailyReport
  db/         pgx connection-per-query execution; DefaultQueries() defines the 3 fixed queries
  logx/       JSONL file logger (one file per day: logs/YYYY-MM-DD.jsonl)
  notifier/   KakaoWork Incoming Webhook client (header+description+divider block format)
  report/     Reads previous day's JSONL and aggregates Stats
  retention/  Deletes JSONL files older than LOG_RETENTION_DAYS
```

**Data flow for keepalive:**
`job` → `db.RunQuery` (per query, with retry) → `logx.Write` (per query + one `keepalive_run` summary) → `retention.Cleanup` → if all failed: `notifier.Send`

**Data flow for daily-report:**
`job` → `report.Build` (reads yesterday's `.jsonl`) → `notifier.Send` → `retention.Cleanup` → `logx.Write`

## Key Design Decisions

- **No connection pool**: `db.RunQuery` opens and closes a new pgx connection per query. Intentional — the program is short-lived and connection-per-run is simpler.
- **Scheduler is external**: The program has no internal timer. Run timing is fully controlled by the server's cron/systemd.
- **Log format drives reporting**: `logx.Record.Event` values (`query_execution`, `keepalive_run`, `daily_report_sent`, `webhook_notification`) are the schema that `report.Build` depends on. Changing event names breaks reporting.
- **Alert threshold**: The webhook alert fires only when all 3 queries fail in a single keepalive run (not per-query).

## Environment Variables

Required (no defaults):
- `DATABASE_URL` — PostgreSQL connection string
- `KAKAOWORK_WEBHOOK_URL` — KakaoWork Incoming Webhook URL

Optional (defaults shown):
- `RUN_MODE=keepalive` — or `daily-report`
- `APP_TIMEZONE=Asia/Seoul`
- `LOG_DIR=./logs`
- `QUERY_TIMEOUT_SECONDS=5`
- `HTTP_TIMEOUT_SECONDS=5`
- `MAX_RETRY_COUNT=1`
- `LOG_RETENTION_DAYS=7`
- `DB_LABEL=gabiadb`

Copy `.env.example` as a reference. `config.Load()` fails fast on missing required values or invalid inputs.
