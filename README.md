# task-planner

Creates distinct Todoist tasks once per configured period. It is for schedules Todoist's built-in recurring tasks cannot express. For normal repeating work, use Todoist's native recurring date instead.

The scheduler is deliberately safe to run repeatedly: each task includes a hidden-looking marker in its description, such as `[task-planner:monthly-finances:2026-08]`. Before creating a task, it searches the selected Todoist project for that marker.

## Setup

1. Install Node.js and pnpm, then run `pnpm install` in this folder.
2. Run `pnpm auth login` from a graphical desktop session. It opens Todoist in your browser, uses OAuth 2.0 authorization code flow with PKCE, and stores the access and refresh tokens in macOS Keychain or Linux Secret Service.
3. Add a Todoist project ID and edit the rule in `rules.json`. After connecting, retrieve your projects with `pnpm auth projects`. The file starts empty, which is safe: it cannot create any task until you add a valid rule.
4. For a one-off local test:

   ```bash
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

Install either the macOS **or** Arch Linux timer—not both. The markers make duplicates unlikely, but a single active scheduler is the intended operational model. Run `pnpm auth login` manually before enabling a timer; background jobs cannot complete a browser login.

### Arch Linux

```bash
cp systemd/task-planner.{service,timer} ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now task-planner.timer
systemctl --user list-timers task-planner.timer
```

`Persistent=true` causes systemd to run a missed timer after login/boot.

### macOS

First run `pnpm auth login` interactively. It stores the OAuth credentials in your login Keychain. Then copy and load the provided agent:

```bash
cp launchd/com.benni.task-planner.plist ~/Library/LaunchAgents/
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.benni.task-planner.plist
launchctl kickstart -k gui/$(id -u)/com.benni.task-planner
```

The launch agent obtains a valid access token from Keychain through `launchd/run-task-planner.zsh`; no credential is written to the plist. Expired access tokens are refreshed automatically, with Todoist's rotated refresh token safely replacing the old one. If pnpm is installed elsewhere, update the `exec` path in that wrapper.
