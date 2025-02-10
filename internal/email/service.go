package email

import (
	"context"
)

type Service interface {
	SendVerification(ctx context.Context, email string, token string) error
	SendPasswordReset(ctx context.Context, email string, token string) error
	SendWelcome(ctx context.Context, email string, name string) error
	SendCustom(ctx context.Context, to string, subject string, content string) error
}
