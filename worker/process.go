package worker

import (
	"context"
	"log"
	"mime"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
	"github.com/rayhankinan/go-imap-notification/util"
)

type Worker struct {
	Ctx       context.Context
	Cancel    context.CancelFunc
	WaitGroup *sync.WaitGroup
	ID        string
}

func NewWorker(ctx context.Context, wg *sync.WaitGroup, id string) *Worker {
	cancelableCtx, cancel := context.WithCancel(ctx)

	return &Worker{
		Ctx:       cancelableCtx,
		Cancel:    cancel,
		WaitGroup: wg,
		ID:        id,
	}
}

func (w *Worker) Fetch(username, password string) ([]string, error) {
	// Create a new connection for each worker
	options := &imapclient.Options{
		WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
	}
	client, err := imapclient.DialTLS("imap.gmail.com:993", options)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// Login to the server
	if err := client.Login(username, password).Wait(); err != nil {
		return nil, err
	}
	defer client.Logout()

	// Select the INBOX mailbox
	// We don't use READ-ONLY mode because we want to mark messages as read
	if _, err := client.Select("INBOX", nil).Wait(); err != nil {
		return nil, err
	}

	// NOTE: This can be removed in order to handle IDLE notifications that are older than 5 minutes
	since := time.Now().Add(-5 * time.Minute)

	// Process the notification by searching for new emails that are not marked as seen
	criteria := &imap.SearchCriteria{
		SentSince: since,
		NotFlag:   []imap.Flag{imap.FlagSeen},
	}
	searchData, err := client.Search(criteria, nil).Wait()
	if err != nil {
		return nil, err
	}

	// Fetch the email messages
	fetchOptions := &imap.FetchOptions{
		Envelope: true,
	}
	messages, err := client.Fetch(searchData.All, fetchOptions).Collect()
	if err != nil {
		return nil, err
	}

	// Mark the messages as seen
	store := &imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Silent: true,
		Flags:  []imap.Flag{imap.FlagSeen},
	}
	if _, err := client.Store(searchData.All, store, nil).Collect(); err != nil {
		return nil, err
	}

	// Loop through the messages
	contents := []string{}
	for _, msg := range messages {
		contents = append(contents, util.PrettifyEnvelope(msg.Envelope))
	}

	return contents, nil
}

func (w *Worker) Start(username, password string, dataChan <-chan *imapclient.UnilateralDataMailbox) {
	// Add a new worker to the WaitGroup
	w.WaitGroup.Add(1)
	defer w.WaitGroup.Done()

	log.Printf("Worker %s has started", w.ID)

	// Handle notifications by listening to the dataChan
	for {
		select {
		case <-w.Ctx.Done():
			log.Printf("Worker %s has been stopped", w.ID)
			return
		case <-dataChan:
			log.Printf("Worker %s has received a new notification", w.ID)

			// Create a new connection for each fetch operation
			contents, err := w.Fetch(username, password)
			if err != nil {
				log.Printf("Worker %s failed to fetch emails: %v", w.ID, err)
				continue
			}

			// Log the contents of the email
			for _, content := range contents {
				log.Printf("Worker %s has processed the email:\n%s", w.ID, content)
			}
		}
	}
}

func (w *Worker) Stop() {
	w.Cancel()
}
