package ports

import "context"

// EmailMessage is the data needed to send a single transactional email.
type EmailMessage struct {
	To      string
	Subject string
	Body    string // plain text
}

// EmailSender abstracts outgoing email delivery.
// The SMTP implementation is used for both local development (Mailpit) and
// production (any SMTP relay such as SendGrid's SMTP API).
type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) error
}
