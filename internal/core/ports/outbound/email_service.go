package outbound

import "context"

type EmailService interface {
	SendVerificationEmail(ctx context.Context, email, code string) error
}
