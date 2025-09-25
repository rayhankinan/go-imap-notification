package listener

import (
	"encoding/json"
	"fmt"
	"log"
	"mime"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
	"github.com/hibiken/asynq"
	"github.com/rayhankinan/go-imap-notification/messages"
)

type Listener struct {
	Client  *imapclient.Client
	IdleCmd *imapclient.IdleCommand
}

func NewListener(username, password string, asynqClient *asynq.Client) (*Listener, error) {
	// Listen for notifications of mailbox changes
	options := &imapclient.Options{
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			Mailbox: func(data *imapclient.UnilateralDataMailbox) {
				payload, err := json.Marshal(messages.EmailMessage{
					Username:    username,
					Password:    password,
					NumMessages: data.NumMessages,
				})
				if err != nil {
					log.Printf("Failed to marshal email message: %v\n", err)
					return
				}

				task := asynq.NewTask(messages.TypeEmailNotify, payload)
				info, err := asynqClient.Enqueue(task)
				if err != nil {
					log.Printf("Failed to enqueue task: %v\n", err)
					return
				}

				log.Printf("Enqueued task: id=%s queue=%s\n", info.ID, info.Queue)
			},
		},
		WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
	}
	client, err := imapclient.DialTLS("imap.gmail.com:993", options)
	if err != nil {
		return nil, err
	}

	// Login to the server
	if err := client.Login(username, password).Wait(); err != nil {
		return nil, err
	}

	// Select the INBOX mailbox
	// We use READ-ONLY mode to avoid accidentally marking emails as read
	selectOptions := &imap.SelectOptions{
		ReadOnly: true,
	}
	if _, err := client.Select("INBOX", selectOptions).Wait(); err != nil {
		return nil, err
	}

	// IDLE command will block until a new email arrives
	idleCmd, err := client.Idle()
	if err != nil {
		return nil, err
	}

	return &Listener{
		Client:  client,
		IdleCmd: idleCmd,
	}, nil
}

func (l *Listener) Stop() error {
	if err := l.IdleCmd.Close(); err != nil {
		return fmt.Errorf("failed to close IDLE command: %w", err)
	}
	if err := l.Client.Logout().Wait(); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}
	if err := l.Client.Close(); err != nil {
		return fmt.Errorf("failed to close client: %w", err)
	}
	return nil
}
