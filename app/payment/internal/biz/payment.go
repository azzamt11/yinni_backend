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
	Method string
}

// PaymentRepo is a Greater repo.
type PaymentRepo interface {
	Pay(context.Context, *Payment) (*Payment, error)
}

// PaymentUsecase is a Payment usecase.
type PaymentUsecase struct {
	repo PaymentRepo
	log  *log.Helper
}

// NewPaymentUsecase new a Payment usecase.
func NewPaymentUsecase(repo PaymentRepo, logger log.Logger) *PaymentUsecase {
	return &PaymentUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// CreateProduct creates a new Product.
func (uc *PaymentUsecase) Pay(ctx context.Context, p *Payment) (*Payment, error) {
	uc.log.Infof("CreateProduct: %v", p.Method)
	return uc.repo.Pay(ctx, p)
}
