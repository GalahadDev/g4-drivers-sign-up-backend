package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"g4-services/api/config"
)

const brevoEndpoint = "https://api.brevo.com/v3/smtp/email"

// brevoClient has a bounded timeout so fire-and-forget email goroutines can't
// leak if Brevo stops responding (audit A3).
var brevoClient = &http.Client{Timeout: 15 * time.Second}

type brevoRecipient struct {
	Email string `json:"email"`
}

type brevoSender struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type brevoPayload struct {
	Sender      brevoSender      `json:"sender"`
	To          []brevoRecipient `json:"to"`
	Subject     string           `json:"subject"`
	HTMLContent string           `json:"htmlContent"`
}

// SendEmail sends an HTML email to one or more recipients via the Brevo API.
// Returns nil without sending if BREVO_API_KEY is not configured.
func SendEmail(to []string, subject string, bodyHTML string, cfg *config.AppConfig) error {
	if cfg.BrevoAPIKey == "" {
		slog.Warn("BREVO_API_KEY not set, email not sent", "subject", subject)
		return nil
	}

	recipients := make([]brevoRecipient, len(to))
	for i, addr := range to {
		recipients[i] = brevoRecipient{Email: addr}
	}

	payload := brevoPayload{
		Sender:      brevoSender{Name: "G4 Car Service", Email: cfg.BrevoFromEmail},
		To:          recipients,
		Subject:     subject,
		HTMLContent: bodyHTML,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("email: marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, brevoEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("email: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", cfg.BrevoAPIKey)

	resp, err := brevoClient.Do(req)
	if err != nil {
		return fmt.Errorf("email: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		slog.Error("Brevo rejected email", "status", resp.StatusCode, "body", string(respBody), "subject", subject)
		return fmt.Errorf("email: Brevo returned status %d", resp.StatusCode)
	}

	slog.Info("Email sent via Brevo", "subject", subject, "recipients", len(to))
	return nil
}
