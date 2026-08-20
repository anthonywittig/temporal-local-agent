# temporal-local-agent

An AI chat agent whose conversation state is managed by [Temporal](https://temporal.io),
answering with a light model running locally via [Ollama](https://ollama.com).

## How it works

Each chat session is one long-running Temporal workflow (`internal/chat/workflow.go`):

- The conversation history is **workflow state** — durable, queryable, and it
  survives worker restarts and crashes.
- Each user message is a **workflow update** (`send-message`). The handler
  appends the message to history, runs the `CompleteChat` **activity** (which
  calls Ollama's `/api/chat`), appends the reply, and returns it as the update
  result. Activity retries come for free from Temporal.
- When Temporal suggests the event history is getting large, the workflow
  **continues-as-new**, carrying the conversation forward into a fresh run.
- A `history` query exposes the transcript; an `end-chat` signal closes the
  session.

## Prerequisites

- Go 1.22+
- [Temporal CLI](https://docs.temporal.io/cli) (`brew install temporal`)
- [Ollama](https://ollama.com) (`brew install ollama`), with a model pulled:
  `ollama pull llama3.2`

## Running

```sh
make start    # Temporal dev server + Ollama + worker (idempotent; pulls the model if missing)
make status   # show what's running
make stop     # stop what `make start` started
make destroy  # stop everything and delete local state (chat history!)
```

`make start` records PID files under `.run/` and only ever manages processes
it started — a Temporal server or Ollama you're already running elsewhere is
detected and left alone. Logs land in `.run/*.log`; the Temporal UI is at
http://localhost:8233.

If you'd rather run the pieces by hand, it's three long-lived processes:
`temporal server start-dev --db-filename .temporal.db`, `ollama serve`, and
`go run ./cmd/worker`.

Then chat:

```sh
go run ./cmd/chat                   # interactive REPL, default session
go run ./cmd/chat -session work     # a separate named session
go run ./cmd/chat -m "hello"        # one-shot message
go run ./cmd/chat -end              # end the session workflow
```

Quitting the REPL (`/quit`) leaves the workflow running — rejoin the same
session later and the model still has the full conversation context. Watch the
workflow's event history live in the Temporal UI at http://localhost:8233.

Set `OLLAMA_MODEL` on the worker to use a different model (default `llama3.2`).
