package worker

import (
	"log"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/rayhankinan/go-imap-notification/util"
)

type Worker struct {
	Username string
	Password string
	DataChan <-chan *imapclient.UnilateralDataMailbox
	Duration time.Duration
}

func NewWorker(username, password string, dataChan <-chan *imapclient.UnilateralDataMailbox, duration time.Duration) *Worker {
	return &Worker{
		Username: username,
		Password: password,
		DataChan: dataChan,
		Duration: duration,
	}
}

func (w *Worker) Start(id string, wg *sync.WaitGroup) {
	// Add a new worker to the WaitGroup
	wg.Add(1)
	defer wg.Done()

	log.Printf("Worker %s has started", id)

	// Handle notifications by listening to the dataChan
	for range w.DataChan {
		log.Printf("Worker %s has received a new message events", id)

		// Determine the time to fetch emails
		sentSince := time.Now().Add(-w.Duration)

		// Create a new connection for each fetch operation
		contents, err := util.FetchMails(w.Username, w.Password, sentSince)
		if err != nil {
			log.Printf("Worker %s failed to fetch emails: \"%v\"", id, err)
			continue
		}

		// Log the contents of the email
		for _, content := range contents {
			log.Printf("Worker %s has processed the email:\n%s", id, content)
		}
	}

	log.Printf("Worker %s has finished processing", id)
}
