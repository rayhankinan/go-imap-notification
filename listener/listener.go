package listener

import (
	"log"
	"mime"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
)

type Listener struct {
	Client  *imapclient.Client
	IdleCmd *imapclient.IdleCommand
}

func NewListener(username, password string, dataChan chan<- *imapclient.UnilateralDataMailbox) (*Listener, error) {
	// Listen for notifications of mailbox changes
	options := &imapclient.Options{
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			Mailbox: func(data *imapclient.UnilateralDataMailbox) {
				// Check if the mailbox data is nil
				if data == nil {
					log.Println("No data received")
					return
				}

				// Check if the mailbox is nil
				if data.NumMessages == nil {
					log.Println("No messages in mailbox")
					return
				}

				// Send the mailbox event to the data channel
				dataChan <- data
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
	return l.IdleCmd.Close()
}

func (l *Listener) Logout() error {
	return l.Client.Logout().Wait()
}

func (l *Listener) Close() error {
	return l.Client.Close()
}
