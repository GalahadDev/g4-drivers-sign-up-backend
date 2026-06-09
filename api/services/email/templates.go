package email

import (
	"fmt"
	"time"
)

// WelcomeTemplate returns the HTML welcome email for a newly registered driver.
func WelcomeTemplate(fullName string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="margin:0;padding:0;font-family:Arial,sans-serif;background-color:#f4f4f4;">
  <div style="max-width:600px;margin:0 auto;background-color:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 4px 6px rgba(0,0,0,0.1);">
    <div style="background-color:#2E4A35;padding:20px;text-align:center;">
      <h1 style="color:#D4AF37;margin:0;font-size:24px;">G4 Car Service Inc</h1>
    </div>
    <div style="padding:30px;color:#333333;">
      <h2 style="color:#2E4A35;">Welcome, %s!</h2>
      <p>Thank you for registering with G4 Car Service.</p>
      <p>Your application has been received and is currently being processed by our team.</p>
      <p>We will reach out to you once the onboarding process is complete.</p>
      <div style="margin-top:30px;padding:15px;background-color:#f9f9f9;border-left:4px solid #D4AF37;">
        <p style="margin:0;font-size:14px;color:#555;">Status: <strong style="color:#2E4A35;">Submitted</strong></p>
      </div>
    </div>
    <div style="background-color:#2E4A35;padding:15px;text-align:center;color:#D4AF37;font-size:12px;">
      &copy; 2026 G4 Car Service Inc. All rights reserved.
    </div>
  </div>
</body>
</html>`, fullName)
}

// AdminNotifData holds the driver details included in the admin notification email.
type AdminNotifData struct {
	FullName     string
	Category     string // "comfort" or "luxury"
	Phone        string
	RegisteredAt time.Time
	DashboardURL string
}

// AdminNotificationTemplate returns the HTML admin notification email for a new driver registration.
func AdminNotificationTemplate(d AdminNotifData) string {
	categoryLabel := "Comfort"
	categoryColor := "#4A90D9"
	if d.Category == "luxury" {
		categoryLabel = "Luxury"
		categoryColor = "#D4AF37"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="margin:0;padding:0;font-family:Arial,sans-serif;background-color:#f4f4f4;">
  <div style="max-width:600px;margin:0 auto;background-color:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 4px 6px rgba(0,0,0,0.1);">
    <div style="background-color:#2E4A35;padding:20px;text-align:center;">
      <h1 style="color:#D4AF37;margin:0;font-size:22px;">G4 Car Service — Admin</h1>
    </div>
    <div style="padding:30px;color:#333333;">
      <h2 style="color:#2E4A35;margin-top:0;">New Driver Registration</h2>
      <p style="color:#555;margin-bottom:24px;">A driver has completed the full registration flow.</p>
      <table style="width:100%%;border-collapse:collapse;">
        <tr>
          <td style="padding:10px 0;border-bottom:1px solid #eee;color:#888;width:40%%;">Name</td>
          <td style="padding:10px 0;border-bottom:1px solid #eee;font-weight:bold;">%s</td>
        </tr>
        <tr>
          <td style="padding:10px 0;border-bottom:1px solid #eee;color:#888;">Category</td>
          <td style="padding:10px 0;border-bottom:1px solid #eee;">
            <span style="background-color:%s;color:#fff;padding:2px 10px;border-radius:12px;font-size:13px;">%s</span>
          </td>
        </tr>
        <tr>
          <td style="padding:10px 0;border-bottom:1px solid #eee;color:#888;">Phone</td>
          <td style="padding:10px 0;border-bottom:1px solid #eee;">%s</td>
        </tr>
        <tr>
          <td style="padding:10px 0;color:#888;">Registered At</td>
          <td style="padding:10px 0;">%s</td>
        </tr>
      </table>
      <div style="margin-top:32px;text-align:center;">
        <a href="%s" style="background-color:#2E4A35;color:#D4AF37;padding:12px 28px;border-radius:6px;text-decoration:none;font-weight:bold;font-size:15px;">
          Review in Dashboard
        </a>
      </div>
    </div>
    <div style="background-color:#2E4A35;padding:15px;text-align:center;color:#D4AF37;font-size:12px;">
      &copy; 2026 G4 Car Service Inc. All rights reserved.
    </div>
  </div>
</body>
</html>`,
		d.FullName,
		categoryColor, categoryLabel,
		d.Phone,
		d.RegisteredAt.Format("Jan 02, 2006 at 3:04 PM"),
		d.DashboardURL,
	)
}
