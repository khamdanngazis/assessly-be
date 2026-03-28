package email

import (
	"bytes"
	"fmt"
	"html/template"

	"gopkg.in/gomail.v2"
)

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// EmailSender handles email sending via SMTP
type EmailSender struct {
	config SMTPConfig
	dialer *gomail.Dialer
}

// NewEmailSender creates a new email sender
func NewEmailSender(config SMTPConfig) *EmailSender {
	dialer := gomail.NewDialer(config.Host, config.Port, config.Username, config.Password)
	return &EmailSender{
		config: config,
		dialer: dialer,
	}
}

// SendPasswordReset sends a password reset email
func (s *EmailSender) SendPasswordReset(to, resetToken string, resetURL string) error {
	subject := "Password Reset Request"
	
	// Construct reset link
	resetLink := fmt.Sprintf("%s?token=%s", resetURL, resetToken)
	
	// Email template
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Reset</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2>Password Reset Request</h2>
        <p>You requested to reset your password. Click the button below to proceed:</p>
        <div style="margin: 30px 0;">
            <a href="{{.ResetLink}}" 
               style="background-color: #007bff; color: white; padding: 12px 24px; 
                      text-decoration: none; border-radius: 4px; display: inline-block;">
                Reset Password
            </a>
        </div>
        <p>Or copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #666;">{{.ResetLink}}</p>
        <p style="margin-top: 30px; color: #666; font-size: 0.9em;">
            This link will expire in 1 hour. If you didn't request this, please ignore this email.
        </p>
    </div>
</body>
</html>
`

	t, err := template.New("reset").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	var body bytes.Buffer
	if err := t.Execute(&body, map[string]string{"ResetLink": resetLink}); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	return s.send(to, subject, body.String())
}

// SendTestAccessToken sends test access token to participant
func (s *EmailSender) SendTestAccessToken(to, testTitle, accessToken, accessURL string) error {
	subject := fmt.Sprintf("Access Link for Test: %s", testTitle)
	
	// Construct access link
	accessLink := fmt.Sprintf("%s?token=%s", accessURL, accessToken)
	
	// Email template
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Test Access</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2>Test Access Granted</h2>
        <p>You have been granted access to the test: <strong>{{.TestTitle}}</strong></p>
        <p>Click the button below to start the test:</p>
        <div style="margin: 30px 0;">
            <a href="{{.AccessLink}}" 
               style="background-color: #28a745; color: white; padding: 12px 24px; 
                      text-decoration: none; border-radius: 4px; display: inline-block;">
                Start Test
            </a>
        </div>
        <p>Or copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #666;">{{.AccessLink}}</p>
        <p style="margin-top: 30px; color: #666; font-size: 0.9em;">
            This link is unique to you. Do not share it with others.
        </p>
    </div>
</body>
</html>
`

	t, err := template.New("access").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	var body bytes.Buffer
	data := map[string]string{
		"TestTitle":  testTitle,
		"AccessLink": accessLink,
	}
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	return s.send(to, subject, body.String())
}

// SendWelcome sends a welcome email after registration
func (s *EmailSender) SendWelcome(to, name string) error {
	subject := "Welcome to Assessly"
	
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2>Welcome to Assessly!</h2>
        <p>Hi {{.Name}},</p>
        <p>Thank you for signing up. You can now create and manage essay-based assessments.</p>
        <p>If you have any questions, please don't hesitate to reach out.</p>
        <p style="margin-top: 30px;">Best regards,<br>The Assessly Team</p>
    </div>
</body>
</html>
`

	t, err := template.New("welcome").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	var body bytes.Buffer
	if err := t.Execute(&body, map[string]string{"Name": name}); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	return s.send(to, subject, body.String())
}

// send is a helper to send an email with HTML body
func (s *EmailSender) send(to, subject, htmlBody string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.config.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	if err := s.dialer.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email to %s: %w", to, err)
	}

	return nil
}
