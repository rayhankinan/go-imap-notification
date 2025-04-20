package util

import (
	"fmt"
	"mime"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
)

func FetchMails(username, password string, sentSince time.Time) ([]string, error) {
	// Create a new connection for each worker
	options := &imapclient.Options{
		WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
	}
	client, err := imapclient.DialTLS("imap.gmail.com:993", options)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	defer client.Close()

	// Login to the server
	if err := client.Login(username, password).Wait(); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	defer client.Logout()

	// Select the INBOX mailbox
	// We don't use READ-ONLY mode because we want to mark messages as read
	if _, err := client.Select("INBOX", nil).Wait(); err != nil {
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}

	// Process the notification by searching for new emails that are not marked as seen
	criteria := &imap.SearchCriteria{
		SentSince: sentSince,
		NotFlag:   []imap.Flag{imap.FlagSeen},
	}
	searchData, err := client.Search(criteria, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to search for emails: %w", err)
	}

	// Fetch the email messages
	fetchOptions := &imap.FetchOptions{
		Envelope: true,
	}
	messages, err := client.Fetch(searchData.All, fetchOptions).Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch emails: %w", err)
	}

	// Mark the messages as seen
	store := &imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Silent: true,
		Flags:  []imap.Flag{imap.FlagSeen},
	}
	if _, err := client.Store(searchData.All, store, nil).Collect(); err != nil {
		return nil, fmt.Errorf("failed to mark emails as seen: %w", err)
	}

	// Loop through the messages
	contents := []string{}
	for _, msg := range messages {
		contents = append(contents, PrettifyEnvelope(msg.Envelope))
	}

	return contents, nil
}
