package main

import (
	"fmt"
	"log"
	"mime"
	"os"
	"os/signal"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
	"github.com/spf13/cobra"
)

func PrettifyEnvelope(envelope *imap.Envelope) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Message-ID: %s\n", envelope.MessageID))
	sb.WriteString(fmt.Sprintf("Date: %s\n", envelope.Date))
	sb.WriteString(fmt.Sprintf("Subject: %s\n", envelope.Subject))

	fromAddresses := []string{}
	for _, from := range envelope.From {
		fromAddresses = append(fromAddresses, from.Addr())
	}
	if len(fromAddresses) > 0 {
		sb.WriteString(fmt.Sprintf("From: %s\n", strings.Join(fromAddresses, ", ")))
	}

	senderAddresses := []string{}
	for _, sender := range envelope.Sender {
		senderAddresses = append(senderAddresses, sender.Addr())
	}
	if len(senderAddresses) > 0 {
		sb.WriteString(fmt.Sprintf("Sender: %s\n", strings.Join(senderAddresses, ", ")))
	}

	replyToAddresses := []string{}
	for _, replyTo := range envelope.ReplyTo {
		replyToAddresses = append(replyToAddresses, replyTo.Addr())
	}
	if len(replyToAddresses) > 0 {
		sb.WriteString(fmt.Sprintf("Reply-To: %s\n", strings.Join(replyToAddresses, ", ")))
	}

	toAddresses := []string{}
	for _, to := range envelope.To {
		toAddresses = append(toAddresses, to.Addr())
	}
	if len(toAddresses) > 0 {
		sb.WriteString(fmt.Sprintf("To: %s\n", strings.Join(toAddresses, ", ")))
	}

	ccAddresses := []string{}
	for _, cc := range envelope.Cc {
		ccAddresses = append(ccAddresses, cc.Addr())
	}
	if len(ccAddresses) > 0 {
		sb.WriteString(fmt.Sprintf("Cc: %s\n", strings.Join(ccAddresses, ", ")))
	}

	bccAddresses := []string{}
	for _, bcc := range envelope.Bcc {
		bccAddresses = append(bccAddresses, bcc.Addr())
	}
	if len(bccAddresses) > 0 {
		sb.WriteString(fmt.Sprintf("Bcc: %s\n", strings.Join(bccAddresses, ", ")))
	}

	inReplyTo := envelope.InReplyTo
	if len(inReplyTo) > 0 {
		sb.WriteString(fmt.Sprintf("In-Reply-To: %s\n", strings.Join(inReplyTo, ", ")))
	}

	return sb.String()
}

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

				dataChan := make(chan *imapclient.UnilateralDataMailbox, 1)
				options := &imapclient.Options{
					UnilateralDataHandler: &imapclient.UnilateralDataHandler{
						Mailbox: func(data *imapclient.UnilateralDataMailbox) {
							dataChan <- data
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

				quitChan := make(chan os.Signal, 1)
				signal.Notify(quitChan, os.Interrupt)

				var idleCmd *imapclient.IdleCommand
				if idleCmd, err = client.Idle(); err != nil {
					log.Fatalf("IDLE command failed: %v", err)
				}

				for {
					select {
					case data := <-dataChan:
						if data.NumMessages == nil {
							log.Println("Mailbox data does not provide number of messages")
							continue
						}

						seqNum := *data.NumMessages

						log.Printf("New email arrived: %d\n", seqNum)

						if err := idleCmd.Close(); err != nil {
							log.Fatalf("IDLE command close failed: %v", err)
						}

						numSet := imap.SeqSetNum(seqNum)
						fetchOptions := &imap.FetchOptions{
							Envelope: true,
						}
						fetchCmd := client.Fetch(numSet, fetchOptions)

						messages, err := fetchCmd.Collect()
						if err != nil {
							log.Fatalf("Fetch failed: %v", err)
						}

						for _, msg := range messages {
							log.Printf("Email details:\n%s", PrettifyEnvelope(msg.Envelope))
						}

						if idleCmd, err = client.Idle(); err != nil {
							log.Fatalf("IDLE command failed: %v", err)
						}

					case <-quitChan:
						log.Println("Exiting...")

						if err := idleCmd.Close(); err != nil {
							log.Fatalf("IDLE command close failed: %v", err)
						}

						return
					}
				}
			},
		},
	)

	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
