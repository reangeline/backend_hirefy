package email

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

//go:embed templates/verification_email.html
var verificationEmailTemplate string

type emailServiceImpl struct {
	client    *sesv2.Client
	fromEmail string
	fromName  string
	isDev     bool
}

type EmailConfig struct {
	Client    *sesv2.Client
	FromEmail string
	FromName  string
	IsDev     bool
}

func NewEmailService(config EmailConfig) outbound.EmailService {
	return &emailServiceImpl{
		client:    config.Client,
		fromEmail: config.FromEmail,
		fromName:  config.FromName,
		isDev:     config.IsDev,
	}
}

type VerificationEmailData struct {
	Name string
	Code string
}

func (s *emailServiceImpl) SendVerificationEmail(ctx context.Context, email, code string) error {
	// Em desenvolvimento, mostrar no log
	if s.isDev {
		fmt.Printf("%s", "\n"+strings.Repeat("=", 60)+"\n")
		fmt.Printf("📧 VERIFICATION EMAIL (DEV MODE)\n")
		fmt.Printf("%s", strings.Repeat("=", 60)+"\n")
		fmt.Printf("To: %s\n", email)
		fmt.Printf("From: %s <%s>\n", s.fromName, s.fromEmail)
		fmt.Printf("Subject: Verify your ApplyWise account\n")
		fmt.Printf("%s", strings.Repeat("-", 60)+"\n")
		fmt.Printf("Your verification code is:\n\n")
		fmt.Printf("    %s\n\n", code)
		fmt.Printf("This code expires in 15 minutes.\n")
		fmt.Printf("%s", strings.Repeat("=", 60)+"\n\n")
	}

	// Sempre enviar via SES (produção ou dev)
	return s.sendViaSES(ctx, email, code)
}

func (s *emailServiceImpl) sendViaSES(ctx context.Context, toEmail, code string) error {
	// ✅ CORRIGIR: Verificar se template foi carregado
	if verificationEmailTemplate == "" {
		fmt.Printf("⚠️ Template vazio, usando fallback HTML\n")
		verificationEmailTemplate = getFallbackTemplate()
	}

	// Parse template
	tmpl, err := template.New("verification").Parse(verificationEmailTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	// Dados do template
	data := VerificationEmailData{
		Name: extractNameFromEmail(toEmail),
		Code: code,
	}

	// Renderizar HTML
	var htmlBody bytes.Buffer
	if err := tmpl.Execute(&htmlBody, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// ✅ DEBUG: Mostrar preview do HTML
	fmt.Printf("📧 Sending email to: %s\n", toEmail)
	fmt.Printf("📝 HTML Body length: %d bytes\n", htmlBody.Len())
	fmt.Printf("🔑 Code: %s\n", code)

	// Criar texto plano como fallback
	textBody := fmt.Sprintf(`
Hi %s,

Thanks for signing up for ApplyWise!

Your verification code is: %s

This code expires in 15 minutes.

If you didn't create an ApplyWise account, please ignore this email.

Need help? Contact us at support@applywise.com

© 2026 ApplyWise. All rights reserved.
`, data.Name, code)

	// Criar mensagem
	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail)),
		Destination: &types.Destination{
			ToAddresses: []string{toEmail},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data:    aws.String("Verify your ApplyWise account"),
					Charset: aws.String("UTF-8"),
				},
				Body: &types.Body{
					Html: &types.Content{
						Data:    aws.String(htmlBody.String()),
						Charset: aws.String("UTF-8"),
					},
					Text: &types.Content{
						Data:    aws.String(textBody),
						Charset: aws.String("UTF-8"),
					},
				},
			},
		},
	}

	// Enviar email
	result, err := s.client.SendEmail(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send email via SES: %w", err)
	}

	fmt.Printf("✅ Email sent successfully! MessageId: %s\n", *result.MessageId)
	return nil
}

// extractNameFromEmail extrai nome do email
func extractNameFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 0 {
		return "there"
	}

	name := strings.Split(parts[0], ".")[0]
	if name == "" {
		return "there"
	}

	// Capitalizar primeira letra
	return strings.Title(strings.ToLower(name))
}

// getFallbackTemplate retorna template inline se embed falhar
func getFallbackTemplate() string {
	return `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Verify Your Email</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f4f4f4;
        }
        .container {
            background-color: #ffffff;
            border-radius: 8px;
            padding: 40px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .logo {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo h1 {
            color: #0D9488;
            font-size: 28px;
            margin: 0;
        }
        .code-container {
            background-color: #F3F4F6;
            border-radius: 8px;
            padding: 20px;
            text-align: center;
            margin: 30px 0;
        }
        .code {
            font-size: 32px;
            font-weight: bold;
            letter-spacing: 8px;
            color: #0D9488;
            font-family: 'Courier New', monospace;
        }
        .footer {
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #E5E7EB;
            font-size: 14px;
            color: #6B7280;
            text-align: center;
        }
        .warning {
            background-color: #FEF3C7;
            border-left: 4px solid #F59E0B;
            padding: 12px;
            margin: 20px 0;
            border-radius: 4px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo">
            <h1>🎯 ApplyWise</h1>
        </div>
        
        <h2>Verify Your Email Address</h2>
        
        <p>Hi {{.Name}},</p>
        
        <p>Thanks for signing up for ApplyWise! To complete your registration and unlock all features, please verify your email address.</p>
        
        <div class="code-container">
            <p style="margin: 0; font-size: 14px; color: #6B7280;">Your verification code is:</p>
            <div class="code">{{.Code}}</div>
        </div>
        
        <p style="text-align: center;">
            <strong>This code expires in 15 minutes.</strong>
        </p>
        
        <div class="warning">
            <strong>⚠️ Security Notice:</strong> If you didn't create an ApplyWise account, please ignore this email.
        </div>
        
        <div class="footer">
            <p>Need help? Contact us at support@applywise.com</p>
            <p>&copy; 2026 ApplyWise. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`
}
