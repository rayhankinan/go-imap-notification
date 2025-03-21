package main

import (
	"context"
	"fmt"
	"log"
	"mime"
	"os"
	"os/signal"
	"sync"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
	"github.com/rayhankinan/go-imap-notification/worker"
	"github.com/spf13/cobra"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	cli := &cobra.Command{}

	cli.AddCommand(
		&cobra.Command{
			Use:   "notify-email",
			Short: "Notify when a new email arrives",
			Long:  "Notify when a new email arrives",
			Run: func(cmd *cobra.Command, args []string) {
				if len(args) != 2 {
					log.Fatal("Usage: notify-email <username> <password>")
				}

				username := args[0]
				password := args[1]

				// Create a channel to receive notifications
				bufferSize := 1
				dataChan := make(chan *imapclient.UnilateralDataMailbox, bufferSize)

				// Listen for notifications of mailbox changes
				options := &imapclient.Options{
					UnilateralDataHandler: &imapclient.UnilateralDataHandler{
						Mailbox: func(data *imapclient.UnilateralDataMailbox) {
							select {
							case dataChan <- data:
								log.Println("A new notification has been received")
							default:
								log.Println("Current notification has not been processed yet")
							}
						},
					},
					WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
				}
				client, err := imapclient.DialTLS("imap.gmail.com:993", options)
				if err != nil {
					log.Fatalf("DialTLS failed: %v", err)
				}
				defer client.Close()

				// Login to the server
				if err := client.Login(username, password).Wait(); err != nil {
					log.Fatalf("Login failed: %v", err)
				}
				defer client.Logout()

				// Select the INBOX mailbox
				// We use READ-ONLY mode to avoid accidentally marking emails as read
				selectOptions := &imap.SelectOptions{
					ReadOnly: true,
				}
				if _, err := client.Select("INBOX", selectOptions).Wait(); err != nil {
					log.Fatal(err)
				}

				log.Println("Waiting for new emails...")

				// IDLE command will block until a new email arrives
				idleCmd, err := client.Idle()
				if err != nil {
					log.Fatalf("IDLE command failed: %v", err)
				}

				// Create a context to handle signals
				ctx := cmd.Context()
				waitGroup := &sync.WaitGroup{}

				// Spawn workers to process the notification
				numWorker := 5
				workers := []*worker.Worker{}
				for i := range numWorker {
					w := worker.NewWorker(ctx, waitGroup, fmt.Sprintf("worker-%d", i+1))
					workers = append(workers, w)

					go w.Start(username, password, dataChan)
				}

				// Wait for a signal to exit
				quitChan := make(chan os.Signal, 1)
				signal.Notify(quitChan, os.Interrupt)
				<-quitChan

				// Stop the IDLE command
				if err := idleCmd.Close(); err != nil {
					log.Fatalf("IDLE command close failed: %v", err)
				}

				// Cancel the context to stop the workers
				for _, w := range workers {
					w.Cancel()
				}

				// Wait for all workers to finish
				waitGroup.Wait()

				log.Println("Exiting...")
			},
		},
	)

	if err := cli.ExecuteContext(context.Background()); err != nil {
		log.Fatal(err)
	}
}
