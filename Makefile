.PHONY: start stop destroy status chat build

## start: launch Temporal dev server, Ollama, and the worker (idempotent)
start:
	./scripts/start.sh

## stop: stop the processes started by `make start`
stop:
	./scripts/stop.sh

## destroy: stop everything and delete local state (chat history!)
destroy:
	./scripts/destroy.sh

## status: show what's running
status:
	./scripts/status.sh

## chat: open an interactive chat session
chat:
	go run ./cmd/chat

## build: compile everything
build:
	go build ./...
