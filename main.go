package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/rayhankinan/go-imap-notification/listener"
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

				// Spawn listener to handle notifications
				l, err := listener.NewListener(username, password, dataChan)
				if err != nil {
					log.Fatalf("Failed to create listener: %v", err)
				}
				defer func() {
					if err := l.Stop(); err != nil {
						log.Printf("Failed to stop listener: %v", err)
					}

					if err := l.Logout(); err != nil {
						log.Printf("Failed to logout: %v", err)
					}

					if err := l.Close(); err != nil {
						log.Printf("Failed to close client: %v", err)
					}
				}()

				log.Println("Listening for notifications...")

				// Create a worker to process the notification
				duration := 5 * time.Minute
				w := worker.NewWorker(username, password, dataChan, duration)

				// Spawn workers to process the notification
				numWorker := 5
				waitGroup := &sync.WaitGroup{}
				for i := range numWorker {
					go w.Start(fmt.Sprintf("worker-%d", i+1), waitGroup)
				}

				// Wait for a signal to exit
				quitChan := make(chan os.Signal, 1)
				signal.Notify(quitChan, os.Interrupt)
				<-quitChan

				// Close the data channel
				close(dataChan)

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
