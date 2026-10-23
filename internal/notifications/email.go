package notifications

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"lifelink/internal/models"
)

// SMTPConfig holds optional SMTP settings. Email alerts stay disabled without a host and sender.
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// EmailNotifier sends one private email alert to a set of matched donors.
type EmailNotifier struct {
	config SMTPConfig
}

func NewEmailNotifier(cfg SMTPConfig) *EmailNotifier {
	if strings.TrimSpace(cfg.Port) == "" {
		cfg.Port = "587"
	}
	return &EmailNotifier{config: cfg}
}

func (n *EmailNotifier) Enabled() bool {
	return n != nil && strings.TrimSpace(n.config.Host) != "" && strings.TrimSpace(n.config.From) != ""
}

// SendBloodRequestAlert sends a single email with recipient addresses hidden from one another.
func (n *EmailNotifier) SendBloodRequestAlert(ctx context.Context, recipients []string, req *models.BloodRequest) (int, error) {
	if !n.Enabled() || len(recipients) == 0 {
		return 0, nil
	}
	if (n.config.Username == "") != (n.config.Password == "") {
		return 0, fmt.Errorf("SMTP username and password must both be set or both be empty")
	}

	from, err := mail.ParseAddress(n.config.From)
	if err != nil {
		return 0, fmt.Errorf("invalid SMTP sender address")
	}

	cleanRecipients := make([]string, 0, len(recipients))
	seen := make(map[string]struct{}, len(recipients))
	for _, raw := range recipients {
		address, parseErr := mail.ParseAddress(strings.TrimSpace(raw))
		if parseErr != nil {
			continue
		}
		key := strings.ToLower(address.Address)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		cleanRecipients = append(cleanRecipients, address.Address)
	}
	if len(cleanRecipients) == 0 {
		return 0, nil
	}

	address := net.JoinHostPort(n.config.Host, n.config.Port)
	dialer := net.Dialer{Timeout: 8 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return 0, fmt.Errorf("could not connect to SMTP server: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(12 * time.Second))

	client, err := smtp.NewClient(conn, n.config.Host)
	if err != nil {
		return 0, fmt.Errorf("could not start SMTP session: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); !ok {
		return 0, fmt.Errorf("SMTP server does not support STARTTLS")
	}
	if err := client.StartTLS(&tls.Config{ServerName: n.config.Host, MinVersion: tls.VersionTLS12}); err != nil {
		return 0, fmt.Errorf("could not secure SMTP connection: %w", err)
	}
	if n.config.Username != "" {
		auth := smtp.PlainAuth("", n.config.Username, n.config.Password, n.config.Host)
		if err := client.Auth(auth); err != nil {
			return 0, fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	if err := client.Mail(from.Address); err != nil {
		return 0, fmt.Errorf("SMTP sender was rejected: %w", err)
	}
	for _, recipient := range cleanRecipients {
		if err := client.Rcpt(recipient); err != nil {
			return 0, fmt.Errorf("SMTP recipient was rejected: %w", err)
		}
	}

	body := fmt.Sprintf(
		"LifeLink has a %s blood request near %s.\r\n\r\nPatient/contact name: %s\r\nRequired blood group: %s\r\nHospital: %s\r\nLocation: %s\r\nContact phone: %s\r\n\r\nIf you are available and choose to help, contact the requester directly. This educational matching result is not medical advice; transfusions require clinical cross-matching and medical oversight.\r\n",
		strings.ToUpper(req.Urgency), req.City, req.PatientName, req.BloodGroup, req.Hospital, req.City, req.ContactPhone,
	)
	message := fmt.Sprintf(
		"From: %s\r\nTo: undisclosed-recipients:;\r\nSubject: LifeLink %s blood request\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n%s",
		from.String(), strings.ToLower(req.Urgency), body,
	)

	writer, err := client.Data()
	if err != nil {
		return 0, fmt.Errorf("SMTP server rejected the message: %w", err)
	}
	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return 0, fmt.Errorf("could not write email message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return 0, fmt.Errorf("SMTP server did not accept the email: %w", err)
	}
	_ = client.Quit()

	return len(cleanRecipients), nil
}
