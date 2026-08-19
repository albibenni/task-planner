# task-planner

`task-planner` creates distinct Todoist tasks from shared plans stored in Supabase.
It is a Go CLI with a Bubble Tea TUI: `task-planner add` guides you through task text,
period, due date, priority, and a human-readable Todoist project picker.

Supabase atomically claims each `(task text, period)` pair. You can run the scheduler on
both macOS and Linux without creating duplicate Todoist tasks.

## Install

Install Go 1.26 or newer, then clone and build the project:

```bash
git clone https://github.com/albibenni/task-planner.git
cd task-planner
make build
```

The build writes the local executable to `./task-planner`, which the included macOS and
Linux scheduler files use. To make it available everywhere in your terminal:

```bash
go install ./cmd/task-planner
```

Ensure `$(go env GOPATH)/bin` is on your `PATH`, then confirm:

```bash
task-planner help
```

## First-time setup

1. Create a Supabase project. In **Connect** (or **Database → Connect**), copy the
   **Session Pooler** URL, beginning with `postgres://` or `postgresql://`. Do not use
   `PUBLIC_SUPABASE_URL`, publishable/anon keys, or service-role keys: they are HTTP API
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
4. Add a plan using the guided TUI:

   ```bash
   task-planner add
   ```

   It lists your Todoist projects by name, so you never need to remember project IDs.
5. Review and preview the plans:

   ```bash
   task-planner plans
   task-planner run --dry-run
   ```

## Commands

```text
task-planner config
task-planner auth login
task-planner auth projects
task-planner add
task-planner plans
task-planner delete "Plan the day"
task-planner run [--dry-run]
task-planner completion bash|zsh
```

Task text is the unique plan identifier. Deleting a plan removes it from Supabase, so it
stops scheduling on every configured device.

Enable Bash completion with:

```bash
if command -v task-planner &>/dev/null; then
  eval "$(task-planner completion bash)"
fi
```

Place it in `~/.bashrc` to load it for each terminal.

## Schedulers

Build the local binary with `make build` before enabling a scheduler. Install either the
macOS agent or the systemd user timer—not both unless you intentionally want redundancy.
The shared Supabase claim prevents duplicates.

### Linux with systemd

```bash
cp systemd/task-planner.{service,timer} ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now task-planner.timer
systemctl --user list-timers task-planner.timer
```

### macOS

Update the absolute repository and binary paths in `launchd/run-task-planner.zsh` and
`launchd/com.benni.task-planner.plist`, then:

```bash
cp launchd/com.benni.task-planner.plist ~/Library/LaunchAgents/
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.benni.task-planner.plist
launchctl kickstart -k gui/$(id -u)/com.benni.task-planner
```

## Development and tests

```bash
make fmt
make check
make test-integration
```

`make check` runs `gofmt`, `go vet`, `golangci-lint`, and Go unit tests.
`make test-integration` starts a disposable PostgreSQL 16 Docker container from
`docker-compose.integration.yml`, runs the Supabase-compatible integration tests, then
removes the container and volume.
