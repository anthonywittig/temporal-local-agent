#!/usr/bin/env bash
# Start everything the agent needs: Temporal dev server, Ollama, and the
# worker. Each process gets a PID file and log under .run/. Idempotent —
# anything already running is left alone.
set -euo pipefail
cd "$(dirname "$0")/.."

RUN_DIR=.run
mkdir -p "$RUN_DIR"

MODEL="${OLLAMA_MODEL:-qwen3:14b}"

is_alive() { # is_alive <name> — true if the pidfile points at a live process
  local pid_file="$RUN_DIR/$1.pid"
  [[ -f "$pid_file" ]] && kill -0 "$(cat "$pid_file")" 2>/dev/null
}

start_one() { # start_one <name> <command...>
  local name=$1
  shift
  if is_alive "$name"; then
    echo "$name: already running (pid $(cat "$RUN_DIR/$name.pid"))"
    return
  fi
  nohup "$@" >"$RUN_DIR/$name.log" 2>&1 &
  echo $! >"$RUN_DIR/$name.pid"
  echo "$name: started (pid $!)"
}

wait_for() { # wait_for <label> <timeout_seconds> <check command...>
  local label=$1 timeout=$2
  shift 2
  local waited=0
  until "$@" >/dev/null 2>&1; do
    if ((waited >= timeout)); then
      echo "error: $label not ready after ${timeout}s — check $RUN_DIR/*.log" >&2
      exit 1
    fi
    sleep 1
    ((waited++)) || true
  done
  echo "$label: ready"
}

# --- Temporal dev server ---
if temporal operator cluster health >/dev/null 2>&1; then
  echo "temporal: already running"
else
  start_one temporal temporal server start-dev --db-filename .temporal.db
fi
wait_for temporal 30 temporal operator cluster health

# --- Ollama ---
if curl -sf http://localhost:11434/api/version >/dev/null 2>&1; then
  echo "ollama: already running"
else
  start_one ollama ollama serve
fi
wait_for ollama 30 curl -sf http://localhost:11434/api/version

if ! ollama list | awk '{print $1}' | grep -q "^${MODEL}\(:\|$\)"; then
  echo "pulling model ${MODEL} (this can take a while)..."
  ollama pull "$MODEL"
fi

# --- Worker ---
if ! is_alive worker; then
  go build -o "$RUN_DIR/worker" ./cmd/worker
fi
start_one worker "$RUN_DIR/worker"

echo
echo "all set — chat with: make chat"
