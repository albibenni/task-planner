# task-planner

Creates distinct Todoist tasks once per configured period. It is for schedules Todoist's built-in recurring tasks cannot express. For normal repeating work, use Todoist's native recurring date instead.

The scheduler is deliberately safe across devices. Supabase/Postgres atomically records a claim for each unique task text and period, so a Mac and Linux PC cannot both schedule the same period. Todoist markers remain a recovery safeguard.

## Install

Install a current Node.js release and pnpm, then clone, install, build, and expose the
CLI globally:

```bash
pnpm setup
# Restart your terminal, or run: exec "$SHELL" -l
git clone https://github.com/albibenni/task-planner.git
cd task-planner
pnpm install --frozen-lockfile
pnpm run build
pnpm add --global "$(pwd)"
```

Confirm the installation:

```bash
task-planner help
```

Keep the checkout: the included macOS and Linux scheduler files run the project from
that directory. The global CLI is for interactive commands such as `config`, `add`, and
`plans`.

## Setup

1. Create a Supabase project. In the project dashboard, choose **Connect** (or
   **Database → Connect**) and copy the **Session Pooler** connection string. It must
   begin with `postgres://` or `postgresql://`; the Session Pooler works on IPv4-only
   home networks. Do **not** use `PUBLIC_SUPABASE_URL`, a publishable/anon key, or a
   service-role key: those are HTTP API credentials, not a Postgres connection URL. If
   Supabase shows a password placeholder, replace it with your database password using
   URL encoding when it contains special characters.
2. Run the interactive setup command. It saves the URL in a protected local file and creates the shared database tables:

   ```bash
   task-planner config
   ```

3. Run `task-planner auth login` from a graphical desktop session. It opens Todoist in your browser, uses OAuth 2.0 authorization code flow with PKCE, and stores the access and refresh tokens in macOS Keychain or Linux Secret Service.
4. Retrieve projects, then add a plan once. Task text is the shared unique identifier:

   ```bash
   task-planner auth projects
   task-planner add --text "Plan the day" --period day --project-id YOUR_PROJECT_ID --due today
   ```

5. For a one-off local test:

   ```bash
   pnpm run dry-run
   pnpm run run
   ```

After `task-planner config`, the next commands are:

```bash
task-planner auth login
task-planner auth projects
task-planner add --text "Plan the day" --period day --project-id YOUR_PROJECT_ID --due today
task-planner plans
task-planner run --dry-run
```

Enable the macOS or Linux scheduler only after the dry run looks correct.

## CLI help and completion

The global CLI provides built-in help and shell completions:

```bash
task-planner help
echo 'eval "$(task-planner completion bash)"' >> ~/.bashrc
```

For Zsh, place these lines in `~/.zshrc` after `compinit`:

```zsh
autoload -Uz compinit
compinit
eval "$(task-planner completion zsh)"
```

Run `task-planner config` once on every device. It stores the database URL in
`~/.config/task-planner/environment` with owner-only permissions. Do not commit or share
that file.

## Shared scheduling

The shared database stores active plans and processed periods. A local scheduler first claims `(task text, period key)` using a database primary key. Exactly one device wins; other devices skip the period. Claims left by a crashed process become eligible for recovery after 15 minutes, where Todoist marker detection runs before a new task is created.

Use these commands from any configured device:

```bash
task-planner plans
task-planner delete "Plan the day"
```

Deleting a plan removes it from the shared database and immediately stops it on every device.

## Plans

Plans use `period` (`day`, `week`, or `month`), task text, a Todoist project ID, optional due date, and optional priority. Add them with `task-planner add`; no JSON editing is required.

Plans are strictly validated with Zod before Todoist is contacted. Run `pnpm run check` before committing; it type-checks, formats/lints with Biome, and runs Vitest. Husky runs the same check before every commit.

## Integration tests

Run the Supabase-compatible Postgres integration suite locally with Docker:

```bash
pnpm run test:integration
```

It starts a disposable `postgres:16-alpine` container, runs plan and concurrent-claim tests, then removes the container and its volume.

## Use one scheduler

Install either the macOS **or** Arch Linux timer—not both. The markers make duplicates unlikely, but a single active scheduler is the intended operational model. Run `pnpm auth login` manually before enabling a timer; background jobs cannot complete a browser login.

### Arch Linux

```bash
cp systemd/task-planner.{service,timer} ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now task-planner.timer
systemctl --user list-timers task-planner.timer
```

`Persistent=true` causes systemd to run a missed timer after login/boot.
The service reads `~/.config/task-planner/environment` for `SUPABASE_DB_URL`.

### macOS

First run `pnpm auth login` interactively. It stores the OAuth credentials in your login Keychain. Then copy and load the provided agent:

```bash
cp launchd/com.benni.task-planner.plist ~/Library/LaunchAgents/
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.benni.task-planner.plist
launchctl kickstart -k gui/$(id -u)/com.benni.task-planner
```

The launch agent obtains a valid access token from Keychain through `launchd/run-task-planner.zsh`; no credential is written to the plist. Expired access tokens are refreshed automatically, with Todoist's rotated refresh token safely replacing the old one. If pnpm is installed elsewhere, update the `exec` path in that wrapper.
The wrapper also reads `~/.config/task-planner/environment` for the Supabase connection URL.
