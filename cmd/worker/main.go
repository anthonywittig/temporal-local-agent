// The worker hosts the chat workflow and its activities. Run alongside a
// Temporal server (temporal server start-dev) and Ollama (ollama serve).
package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/anthonywittig/temporal-local-agent/internal/chat"
)

func main() {
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalf("unable to connect to Temporal: %v", err)
	}
	defer c.Close()

	w := worker.New(c, chat.TaskQueue, worker.Options{})
	w.RegisterWorkflow(chat.Workflow)
	w.RegisterActivity(&chat.Activities{})

	log.Printf("worker started on task queue %q", chat.TaskQueue)
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("worker exited: %v", err)
	}
}
