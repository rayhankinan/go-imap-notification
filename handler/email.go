package handler

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rayhankinan/go-imap-notification/messages"
	"github.com/rayhankinan/go-imap-notification/util"
)

const (
	TypeEmailNotify = "email:notify"
)

type Handler interface {
	HandleEmail(ctx context.Context, task *asynq.Task) error
}

type handler struct {
	Duration time.Duration
}

func NewHandler(duration time.Duration) Handler {
	return &handler{
		Duration: duration,
	}
}

func (h *handler) HandleEmail(ctx context.Context, task *asynq.Task) error {
	payloadRaw := task.Payload()

	payload := messages.EmailMessage{}
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return err
	}

	// Determine the time to fetch emails
	sentSince := time.Now().Add(-h.Duration)

	// Create a new connection for each fetch operation
	contents, err := util.FetchMails(payload.Username, payload.Password, sentSince)
	if err != nil {
		return err
	}

	// Log the contents of the email
	for _, content := range contents {
		log.Printf("New email from %s:\n%s", payload.Username, content)
	}

	return nil
}
