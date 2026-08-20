#!/usr/bin/env bash
# Tear everything down: stop all processes, then delete local state — the
# Temporal dev database (all chat history!) and the .run directory.
# The Ollama model stays cached under ~/.ollama.
set -euo pipefail
cd "$(dirname "$0")/.."

./scripts/stop.sh

rm -rf .run .temporal.db
echo "removed .run/ and .temporal.db (chat history is gone)"
