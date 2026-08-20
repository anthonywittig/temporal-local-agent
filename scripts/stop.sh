#!/usr/bin/env bash
# Stop the processes started by start.sh. Only touches PIDs recorded in
# .run/*.pid, so a Temporal server or Ollama you started some other way is
# left alone.
set -euo pipefail
cd "$(dirname "$0")/.."

RUN_DIR=.run

# Stop the worker first so it can shut down cleanly while Temporal is up.
for name in worker temporal ollama; do
  pid_file="$RUN_DIR/$name.pid"
  if [[ ! -f "$pid_file" ]]; then
    echo "$name: not started by us, skipping"
    continue
  fi
  pid=$(cat "$pid_file")
  if kill -0 "$pid" 2>/dev/null; then
    kill "$pid"
    for _ in $(seq 1 20); do
      kill -0 "$pid" 2>/dev/null || break
      sleep 0.5
    done
    if kill -0 "$pid" 2>/dev/null; then
      echo "$name: did not exit, killing (pid $pid)"
      kill -9 "$pid" 2>/dev/null || true
    else
      echo "$name: stopped (pid $pid)"
    fi
  else
    echo "$name: already stopped"
  fi
  rm -f "$pid_file"
done
