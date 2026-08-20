#!/usr/bin/env bash
# Show what's running.
set -euo pipefail
cd "$(dirname "$0")/.."

RUN_DIR=.run

for name in temporal ollama worker; do
  pid_file="$RUN_DIR/$name.pid"
  if [[ -f "$pid_file" ]] && kill -0 "$(cat "$pid_file")" 2>/dev/null; then
    echo "$name: running (pid $(cat "$pid_file"))"
  else
    echo "$name: not running (via $RUN_DIR)"
  fi
done

temporal operator cluster health >/dev/null 2>&1 &&
  echo "temporal endpoint: healthy (UI at http://localhost:8233)" ||
  echo "temporal endpoint: unreachable"
curl -sf http://localhost:11434/api/version >/dev/null 2>&1 &&
  echo "ollama endpoint: healthy" ||
  echo "ollama endpoint: unreachable"
