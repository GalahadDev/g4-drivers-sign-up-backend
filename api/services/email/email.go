package email

import (
	"fmt"
	"g4-services/api/config"
	"log/slog"
	"net/smtp"
)

func SendEmail(to []string, subject string, bodyHTML string, cfg *config.AppConfig) error {
	if cfg.SMTPHost == "" || cfg.SMTPPort == "" {
		slog.Warn("SMTP configuration missing, email not sent", "subject", subject)
		return nil
	}

	auth := smtp.PlainAuth("", cfg.SMTPEmail, cfg.SMTPPassword, cfg.SMTPHost)

	headers := make(map[string]string)
	headers["From"] = cfg.SMTPEmail
	headers["To"] = to[0]
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"UTF-8\""

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + bodyHTML

	addr := fmt.Sprintf("%s:%s", cfg.SMTPHost, cfg.SMTPPort)

	if err := smtp.SendMail(addr, auth, cfg.SMTPEmail, to, []byte(message)); err != nil {
		slog.Error("Failed to send email", "error", err, "to", to)
		return err
	}

	slog.Info("Email sent successfully", "subject", subject, "to", to)
	return nil
}

func GetWelcomeTemplate(fullName string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<body style="margin:0; padding:0; font-family: Arial, sans-serif; background-color: #f4f4f4;">
    <div style="max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 4px 6px rgba(0,0,0,0.1);">
        <!-- Header -->
        <div style="background-color: #2E4A35; padding: 20px; text-align: center;">
            <h1 style="color: #D4AF37; margin: 0; font-size: 24px;">G4 Car Service Inc</h1>
        </div>
        
        <!-- Body -->
        <div style="padding: 30px; color: #333333;">
            <h2 style="color: #2E4A35;">Welcome, %s!</h2>
            <p>Thank you for registering with G4 Car Service.</p>
            <p>Your application has been received and <strong>automatically approved</strong> as part of our initial launch.</p>
            <p>You are now an active driver in our system.</p>
            
            <div style="margin-top: 30px; padding: 15px; background-color: #f9f9f9; border-left: 4px solid #D4AF37;">
                <p style="margin: 0; font-size: 14px; color: #555;">Status: <strong style="color: #2E4A35;">Approved</strong></p>
            </div>
        </div>
        
        <!-- Footer -->
        <div style="background-color: #2E4A35; padding: 15px; text-align: center; color: #D4AF37; font-size: 12px;">
            &copy; 2026 G4 Car Service Inc. All rights reserved.
        </div>
    </div>
</body>
</html>
`, fullName)
}
