# task-planner

`task-planner` creates distinct Todoist tasks from shared plans stored in Supabase.
It is a Go CLI with a Bubble Tea TUI: `task-planner add` guides you through task text,
a date range, recurrence, priority, and a human-readable Todoist project picker.

It creates every Todoist task for the selected date range immediately. Supabase stores the
shared schedule and Todoist task IDs, so every device sees the same schedules and deletion
removes the same tasks.

## Install

Install Go 1.26 or newer, then clone and build the project:

```bash
git clone https://github.com/albibenni/task-planner.git
cd task-planner
make build
make hooks
```

The build writes the local executable to `./task-planner`. `make hooks` enables this
repository's tracked Git hooks in the current clone: formatting and linting run before
commits, while unit tests run before pushes. Run it once after cloning on each development
machine.

To install the Go CLI globally for your current user:

```bash
make install
```

This writes `task-planner` to `$(go env GOPATH)/bin`. Ensure that directory is on your
`PATH`, then confirm:

```bash
task-planner help
```

## First-time setup

1. Create a Supabase project. In **Connect**, open **Direct (Connection string)**, set
   **Connection Method** to **Session pooler**, then set **Type** to **URI** and copy the
   URL (port `5432`), beginning with `postgres://` or `postgresql://`. Do not use the direct
   `db.<project-ref>.supabase.co` URL unless your network supports IPv6 or you pay for
   Supabase's IPv4 add-on: the direct endpoint is IPv6 by default. Do not use
   `PUBLIC_SUPABASE_URL`, publishable/anon keys, or service-role keys; they are HTTP API
   credentials, not a PostgreSQL connection URL.
2. Run the interactive configuration command and paste that URL:

   ```bash
   task-planner config
   ```

   It validates and stores the URL at `~/.config/task-planner/environment` with owner-only
   permissions, and initializes the shared tables.
3. Connect Todoist from a graphical desktop session:

   ```bash
   task-planner auth login
   ```
4. Add a schedule using the guided TUI:

   ```bash
   task-planner add
   ```

   It lists Todoist projects by name, accepts `today`, `tomorrow`, `YYYY-MM-DD`,
   `DD-MM-YYYY`, and `DD/MM/YY` dates, and creates tasks immediately for the inclusive
   range. Choose every day, every other day, or a Monday-first set of weekdays.
   Similar task wording is shown before the final confirmation; choose whether it is a
   duplicate, then confirm the exact range and task count.
5. Review active schedules:

   ```bash
   task-planner plans
   ```

   You can check the local prerequisites at any time with `task-planner status`. It shows
   whether the Supabase database is reachable and the Todoist login is present, and gives
   the missing command.

## Commands

```text
task-planner config
task-planner auth login
task-planner auth projects
task-planner status
task-planner add
task-planner plans
task-planner delete
task-planner completion bash|zsh
```

Task text is the unique plan identifier. `task-planner delete` opens a searchable,
paginated picker of active plans. After confirmation, it removes the plan from Supabase
and deletes the Todoist tasks created by that schedule.

Enable Bash completion with:

```bash
if command -v task-planner &>/dev/null; then
  eval "$(task-planner completion bash)"
fi
```

Place it in `~/.bashrc` to load it for each terminal, then reload the current shell:

```bash
source ~/.bashrc
```

For Zsh, place this in `~/.zshrc` instead and run `source ~/.zshrc`:

```zsh
if command -v task-planner >/dev/null 2>&1; then
  eval "$(task-planner completion zsh)"
fi
```

## No background scheduler required

`task-planner add` creates the complete set of Todoist tasks immediately, so launchd,
systemd, and a permanently running computer are not needed. Existing launchd agents or
systemd timers from older versions should be disabled and removed; this version has no
`task-planner run` command.

## Development and tests

```bash
make fmt
make check
make test-integration
make hooks
```

`make check` runs `gofmt`, `go vet`, `golangci-lint`, and Go unit tests.
`make test-integration` starts a disposable PostgreSQL 16 Docker container from
`docker-compose.integration.yml`, runs the Supabase-compatible integration tests, then
removes the container and volume.
`make hooks` configures this clone to use the tracked `.githooks` scripts.
