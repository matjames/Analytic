package handlers

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

func sendRegistryEmail(to string, subject string, plainBody string) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("recipient email is required")
	}
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if host == "" {
		return fmt.Errorf("SMTP_HOST is not configured")
	}
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if port == "" {
		port = "587"
	}
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if from == "" {
		from = strings.TrimSpace(os.Getenv("SMTP_USER"))
	}
	if from == "" {
		return fmt.Errorf("SMTP_FROM or SMTP_USER must be configured")
	}

	var auth smtp.Auth
	user := strings.TrimSpace(os.Getenv("SMTP_USER"))
	if user != "" {
		auth = smtp.PlainAuth("", user, os.Getenv("SMTP_PASSWORD"), host)
	}

	message := strings.Join([]string{
		"To: " + to,
		"From: " + from,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		plainBody,
	}, "\r\n")
	return smtp.SendMail(host+":"+port, auth, from, []string{to}, []byte(message))
}
