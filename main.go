package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rayhankinan/go-imap-notification/handler"
	"github.com/rayhankinan/go-imap-notification/listener"
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
				if cmd.Flags().NArg() < 2 {
					log.Fatal("Username and password are required")
				}

				username := cmd.Flags().Arg(0)
				password := cmd.Flags().Arg(1)

				// Initialize Asynq
				asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: "localhost:6379"})

				// Spawn listener to handle notifications
				l, err := listener.NewListener(username, password, asynqClient)
				if err != nil {
					log.Fatalf("Failed to create listener: %v", err)
				}
				defer func() {
					if err := l.Stop(); err != nil {
						log.Printf("Failed to stop listener: \"%v\"", err)
					}
				}()

				log.Println("Listening for notifications...")

				// Wait for a signal to exit
				quitChan := make(chan os.Signal, 1)
				signal.Notify(quitChan, os.Interrupt)
				<-quitChan

				log.Println("Exiting...")
			},
		},
		&cobra.Command{
			Use:   "worker",
			Short: "Start the worker to process email notifications",
			Long:  "Start the worker to process email notifications",
			Run: func(cmd *cobra.Command, args []string) {
				// Initialize Asynq server
				srv := asynq.NewServer(
					asynq.RedisClientOpt{Addr: "localhost:6379"},
					asynq.Config{
						Concurrency: 10,
					},
				)

				duration := cmd.Flags().Duration("duration", 5*time.Minute, "Duration in seconds to look back for new emails")

				h := handler.NewHandler(*duration)
				mux := asynq.NewServeMux()
				mux.HandleFunc(handler.TypeEmailNotify, h.HandleEmail)

				if err := srv.Run(mux); err != nil {
					log.Fatalf("Could not run server: %v", err)
				}
			},
		},
	)

	if err := cli.ExecuteContext(context.Background()); err != nil {
		log.Fatal(err)
	}
}
