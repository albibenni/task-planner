#!/bin/zsh
set -euo pipefail

SUPABASE_DB_URL="$(/usr/bin/sed -n 's/^SUPABASE_DB_URL=//p' "$HOME/.config/task-planner/environment")"
export SUPABASE_DB_URL

cd /Users/benni/benni-projects/task-planner
exec /opt/homebrew/bin/pnpm run run
