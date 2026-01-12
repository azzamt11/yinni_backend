package biz

import (
	"context"

	v1 "yinni_backend/api/payment/v1"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	// ErrUserNotFound is user not found.
	ErrUserNotFound = errors.NotFound(v1.ErrorReason_INVALID_PAYMENT.String(), "invalid payment")
)

// Payment is a Payment model.
type Payment struct {
	Hello string
}

// PaymentRepo is a Greater repo.
type PaymentRepo interface {
	Save(context.Context, *Payment) (*Payment, error)
	Update(context.Context, *Payment) (*Payment, error)
	FindByID(context.Context, int64) (*Payment, error)
	ListByHello(context.Context, string) ([]*Payment, error)
	ListAll(context.Context) ([]*Payment, error)
}

// PaymentUsecase is a Payment usecase.
type PaymentUsecase struct {
	repo PaymentRepo
}

// NewPaymentUsecase new a Payment usecase.
func NewPaymentUsecase(repo PaymentRepo) *PaymentUsecase {
	return &PaymentUsecase{repo: repo}
}

// CreatePayment creates a Payment, and returns the new Payment.
func (uc *PaymentUsecase) Pay(ctx context.Context, g *Payment) (*Payment, error) {
	log.Infof("CreatePayment: %v", g.Hello)
	return uc.repo.Save(ctx, g)
}
