package main

import (
	"log"
	"mime"
	"os"
	"os/signal"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
	"github.com/spf13/cobra"
)

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

				options := &imapclient.Options{
					UnilateralDataHandler: &imapclient.UnilateralDataHandler{
						Expunge: func(seqNum uint32) {
							log.Printf("message %v has been expunged", seqNum)
						},
						Mailbox: func(data *imapclient.UnilateralDataMailbox) {
							if data.NumMessages != nil {
								log.Printf("mailbox has %v messages", *data.NumMessages)
							}
						},
						Fetch: func(msg *imapclient.FetchMessageData) {
							log.Printf("message %v has been fetched", msg.SeqNum)
						},
					},
					WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
				}
				client, err := imapclient.DialTLS("imap.gmail.com:993", options)
				if err != nil {
					log.Fatalf("DialTLS failed: %v", err)
				}

				if err := client.Login(args[0], args[1]).Wait(); err != nil {
					log.Fatalf("Login failed: %v", err)
				}
				defer client.Logout()

				if _, err := client.Select("INBOX", &imap.SelectOptions{
					ReadOnly: true,
				}).Wait(); err != nil {
					log.Fatal(err)
				}

				log.Println("Waiting for new emails...")

				idleCmd, err := client.Idle()
				if err != nil {
					log.Fatalf("IDLE command failed: %v", err)
				}

				quitChan := make(chan os.Signal, 1)
				signal.Notify(quitChan, os.Interrupt)
				<-quitChan

				if err := idleCmd.Close(); err != nil {
					log.Fatalf("IDLE command close failed: %v", err)
				}

				log.Println("Exiting...")
			},
		},
	)

	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
