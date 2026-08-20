// The chat CLI starts (or rejoins) a chat session workflow and exchanges
// messages with it. Each message is a workflow update whose result is the
// assistant's reply.
//
// Usage:
//
//	go run ./cmd/chat                     # interactive REPL, default session
//	go run ./cmd/chat -session work       # named session (rejoinable)
//	go run ./cmd/chat -m "hello"          # one-shot message
//	go run ./cmd/chat -end                # end the session workflow
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"

	"github.com/anthonywittig/temporal-local-agent/internal/chat"
)

func main() {
	session := flag.String("session", "default", "chat session name (workflow ID suffix)")
	message := flag.String("m", "", "send a single message and exit")
	end := flag.Bool("end", false, "end the chat session workflow and exit")
	flag.Parse()

	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalf("unable to connect to Temporal: %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	workflowID := "chat-" + *session

	if *end {
		if err := c.SignalWorkflow(ctx, workflowID, "", chat.SignalEndChat, nil); err != nil {
			log.Fatalf("unable to end session: %v", err)
		}
		fmt.Println("session ended.")
		return
	}

	// Start the session workflow, or attach to it if it is already running.
	_, err = c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                chat.TaskQueue,
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}, chat.Workflow, chat.State{})
	if err != nil {
		log.Fatalf("unable to start session: %v", err)
	}

	if *message != "" {
		fmt.Println(send(ctx, c, workflowID, *message))
		return
	}

	fmt.Printf("session %q — type a message, or /quit to leave (session keeps running)\n", *session)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("you> ")
		if !scanner.Scan() {
			break
		}
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		if text == "/quit" {
			break
		}
		fmt.Printf("bot> %s\n\n", send(ctx, c, workflowID, text))
	}
}

func send(ctx context.Context, c client.Client, workflowID, text string) string {
	handle, err := c.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   workflowID,
		UpdateName:   chat.UpdateSendMessage,
		Args:         []interface{}{text},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		log.Fatalf("unable to send message: %v", err)
	}
	var reply string
	if err := handle.Get(ctx, &reply); err != nil {
		log.Fatalf("message failed: %v", err)
	}
	return reply
}
