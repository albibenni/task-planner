# task-planner

Creates distinct Todoist tasks once per configured period. It is for schedules Todoist's built-in recurring tasks cannot express. For normal repeating work, use Todoist's native recurring date instead.

The scheduler is deliberately safe to run repeatedly: each task includes a hidden-looking marker in its description, such as `[task-planner:monthly-finances:2026-08]`. Before creating a task, it searches the selected Todoist project for that marker.

## Setup

1. Install Node.js and pnpm, then run `pnpm install` in this folder.
2. Create a Todoist API token in Todoist: **Settings → Integrations → Developer**.
3. Add a Todoist project ID and edit the rule in `rules.json`. Project IDs can be retrieved with `GET https://api.todoist.com/api/v1/projects` using your token. The file starts empty, which is safe: it cannot create any task until you add a valid rule.
4. Keep the token out of this repository. For a one-off local test:

   ```bash
   export TODOIST_API_TOKEN='…'
   pnpm run dry-run
   pnpm run run
   ```

## Rules

Each rule has `period` (`day`, `week`, or `month`), `content`, `projectId`, optional Todoist `dueString`, and optional priority (`1`–`4`). The process may run at any time; it creates at most one task per rule and period. If a computer is off when a period begins, it creates the missing task at its next run.

```json
{
  "id": "monthly-finances",
  "period": "month",
  "content": "Review monthly finances",
  "projectId": "your-todoist-project-id",
  "dueString": "today",
  "priority": 2
}
```

Rules are strictly validated with Zod before Todoist is contacted. Run `pnpm run check` before committing; it type-checks, formats/lints with Biome, and runs Vitest. Husky runs the same check before every commit.

## Use one scheduler

Install either the macOS **or** Arch Linux timer—not both. The markers make duplicates unlikely, but a single active scheduler is the intended operational model.

### Arch Linux

```bash
mkdir -p ~/.config/task-planner
chmod 700 ~/.config/task-planner
printf 'TODOIST_API_TOKEN=…\n' > ~/.config/task-planner/todoist.env
chmod 600 ~/.config/task-planner/todoist.env
cp systemd/task-planner.{service,timer} ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now task-planner.timer
systemctl --user list-timers task-planner.timer
```

`Persistent=true` causes systemd to run a missed timer after login/boot.

### macOS

Add the token to your login Keychain, then copy and load the provided agent:

```bash
security add-generic-password -a "$USER" -s task-planner.todoist -w '…' -U
cp launchd/com.benni.task-planner.plist ~/Library/LaunchAgents/
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.benni.task-planner.plist
launchctl kickstart -k gui/$(id -u)/com.benni.task-planner
```

The launch agent reads that token from Keychain through `launchd/run-task-planner.zsh`; no token is written to the plist. If pnpm is installed elsewhere, update the `exec` path in that wrapper.
