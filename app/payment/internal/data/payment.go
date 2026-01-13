package data

import (
	"context"

	"yinni_backend/app/payment/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type greeterRepo struct {
	data *Data
	log  *log.Helper
}

// NewPaymentRepo .
func NewPaymentRepo(data *Data, logger log.Logger) biz.PaymentRepo {
	return &greeterRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *greeterRepo) Pay(ctx context.Context, g *biz.Payment) (*biz.Payment, error) {
	return &biz.Payment{
		Method: g.Method,
	}, nil
}
