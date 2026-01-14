package data

import (
	"context"
	"fmt"
	"strings"

	"yinni_backend/app/prompt/internal/biz"

	classifierpb "yinni_backend/api/classifier/v1"
	paymentpb "yinni_backend/api/payment/v1"
	productpb "yinni_backend/api/product/v1"

	"github.com/go-kratos/kratos/v2/log"
)

type promptRepo struct {
	data             *Data
	log              *log.Helper
	productClient    productpb.ProductClient
	paymentClient    paymentpb.PaymentClient
	classifierClient classifierpb.ClassifierClient
}

// NewPromptRepo implements biz.PromptRepo
func NewPromptRepo(
	data *Data,
	logger log.Logger,
	productClient productpb.ProductClient,
	paymentClient paymentpb.PaymentClient,
	classifierClient classifierpb.ClassifierClient,
) biz.PromptRepo {
	return &promptRepo{
		data:             data,
		log:              log.NewHelper(logger),
		productClient:    productClient,
		paymentClient:    paymentClient,
		classifierClient: classifierClient,
	}
}

//
// =======================
// ML CLASSIFICATION
// =======================
//

type mlResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func (r *promptRepo) Classify(ctx context.Context, prompt string) (biz.PromptType, string, error) {
	println("promptRepo.Classify called, prompt =", prompt)

	resp, err := r.classifierClient.Classify(ctx, &classifierpb.ClassifyRequest{
		Text: prompt,
	})
	if err != nil {
		println("gRPC Classify failed:", err.Error())
		return biz.PromptUnknown, "", fmt.Errorf("ml classify failed: %w", err)
	}

	println("gRPC Classify success, type =", resp.Type)

	switch strings.ToLower(resp.Type) {
	case "find_item":
		return biz.PromptFindItem, resp.Value, nil
	case "select_option":
		return biz.PromptSelectOption, resp.Value, nil
	case "make_payment":
		return biz.PromptMakePayment, resp.Value, nil
	default:
		return biz.PromptUnknown, resp.Value, nil
	}
}

//
// =======================
// PRODUCT SERVICE (gRPC)
// =======================
//

func (r *promptRepo) FindItem(ctx context.Context, query string) (map[string]interface{}, error) {

	resp, err := r.productClient.SearchProducts(ctx, &productpb.SearchProductsRequest{
		Query: query,
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"products": resp.Products,
	}, nil
}

func (r *promptRepo) SelectItem(ctx context.Context, id int64) (map[string]interface{}, error) {

	resp, err := r.productClient.GetProduct(ctx, &productpb.GetProductRequest{
		Id: id,
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":    resp.Id,
		"name":  resp.Title,
		"price": resp.ActualPrice,
	}, nil
}

//
// =======================
// PAYMENT SERVICE (gRPC)
// =======================
//

func (r *promptRepo) MakePayment(ctx context.Context, method string) (map[string]interface{}, error) {
	paymentMethod, err := parsePaymentMethod(method)

	resp, err := r.paymentClient.Pay(ctx, &paymentpb.PayRequest{
		Method: paymentMethod,
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":  resp.Status,
		"message": resp.Message,
	}, nil
}

func parsePaymentMethod(method string) (paymentpb.PaymentMethod, error) {
	method = strings.ToUpper(method)

	if v, ok := paymentpb.PaymentMethod_value[method]; ok {
		return paymentpb.PaymentMethod(v), nil
	}

	return paymentpb.PaymentMethod_UNSPECIFIED,
		fmt.Errorf("invalid payment method: %s", method)
}
