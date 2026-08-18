#!/bin/zsh
set -euo pipefail

export TODOIST_API_TOKEN="$(/usr/bin/security find-generic-password -a "$USER" -s task-planner.todoist -w)"
cd /Users/benni/benni-projects/task-planner
exec /opt/homebrew/bin/pnpm run run
