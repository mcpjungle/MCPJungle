#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dashboard_dir="$repo_root/web/dashboard"
build_output_dir="$dashboard_dir/dist"
embedded_output_dir="$repo_root/internal/dashboardui/dist"

cd "$dashboard_dir"

if [[ ! -d node_modules ]]; then
  npm ci
fi

npm run build

rm -rf "$embedded_output_dir"
cp -R "$build_output_dir" "$embedded_output_dir"
