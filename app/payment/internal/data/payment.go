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

func (r *greeterRepo) Save(ctx context.Context, g *biz.Payment) (*biz.Payment, error) {
	return g, nil
}

func (r *greeterRepo) Update(ctx context.Context, g *biz.Payment) (*biz.Payment, error) {
	return g, nil
}

func (r *greeterRepo) FindByID(context.Context, int64) (*biz.Payment, error) {
	return nil, nil
}

func (r *greeterRepo) ListByHello(context.Context, string) ([]*biz.Payment, error) {
	return nil, nil
}

func (r *greeterRepo) ListAll(context.Context) ([]*biz.Payment, error) {
	return nil, nil
}
