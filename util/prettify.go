package util

import (
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
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
