package service

import (
	"context"
	"os"
	"strings"

	v1 "yinni_backend/api/payment/v1"
	"yinni_backend/app/payment/internal/biz"
)

// Map your payment methods to filenames in app/payment/assets
var methodImages = map[string]string{
	"CREDIT_CARD": "visa.svg",
	"DANA":        "dana.png",
	"OVO":         "ovo.jpg",
	"MASTERCARD":  "mastercard.jpg",
	"MANDIRI":     "mandiri.jpg",
	"BRI":         "bri.jpg",
	"VISA":        "visa.svg",
	"SHOPEEPAY":   "shopeepay.png",
	"BCA":         "bca.svg",
}

type PaymentService struct {
	v1.UnimplementedPaymentServer
	uc *biz.PaymentUsecase
	// Suggestion: Add a base URL from config
	baseUrl string
}

func NewPaymentService(uc *biz.PaymentUsecase) *PaymentService {
	baseURL := os.Getenv("PAYMENT_PUBLIC_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8004"
	}

	baseURL = strings.TrimRight(baseURL, "/")

	return &PaymentService{
		uc:      uc,
		baseUrl: baseURL, // Change this based on your environment
	}
}

func (s *PaymentService) Pay(ctx context.Context, in *v1.PayRequest) (*v1.PayReply, error) {
	methodStr := in.Method.String()
	imageName, ok := methodImages[methodStr]
	if !ok {
		imageName = "default.png"
	}

	// Business logic call
	g, err := s.uc.Pay(ctx, &biz.Payment{
		Method: methodStr,
		Image:  imageName,
	})

	if err != nil {
		return nil, err
	}

	// Construct the public URL

	return &v1.PayReply{
		Message: g.Method,
		Status:  "Pending",
		Image:   "/assets/" + imageName,
	}, nil
}
