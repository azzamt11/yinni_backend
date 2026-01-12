package service

import (
	"context"

	v1 "yinni_backend/api/payment/v1"
	"yinni_backend/app/payment/internal/biz"
)

// PaymentService is a greeter service.
type PaymentService struct {
	v1.UnimplementedPaymentServer

	uc *biz.PaymentUsecase
}

// NewPaymentService new a greeter service.
func NewPaymentService(uc *biz.PaymentUsecase) *PaymentService {
	return &PaymentService{uc: uc}
}

// SayHello implements helloworld.PaymentServer.
func (s *PaymentService) SayHello(ctx context.Context, in *v1.PayRequest) (*v1.PayReply, error) {
	g, err := s.uc.Pay(ctx, &biz.Payment{})
	if err != nil {
		return nil, err
	}
	return &v1.PayReply{Message: "Hello " + g.Hello}, nil
}
