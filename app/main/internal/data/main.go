package data

import (
	"context"
	"strings"

	"yinni_backend/app/main/internal/biz"

	// classifypb "yinni_backend/api/classifier/v1"
	// paymentpb "yinni_backend/api/payment/v1"
	// productpb "yinni_backend/api/product/v1"

	"github.com/go-kratos/kratos/v2/log"
)

type mainRepo struct {
	data *Data
	log  *log.Helper
	// productClient  productpb.ProductClient
	// paymentClient  paymentpb.PaymentClient
	// classifyClient classifypb.ClassifierClient
}

// NewPromptRepo implements biz.PromptRepo
func NewMainRepo(
	data *Data,
	logger log.Logger,
	// productClient productpb.ProductClient,
	// paymentClient paymentpb.PaymentClient,
) biz.MainRepo {
	return &mainRepo{
		data: data,
		log:  log.NewHelper(logger),
		// productClient: productClient,
		// paymentClient: paymentClient,
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

func (r *mainRepo) Classify(ctx context.Context, prompt string) (biz.PromptType, string, error) {
	println("promptRepo.Classify called, prompt =", prompt)
	// if err != nil {
	// 	println("gRPC Classify failed:", err.Error())
	// 	return biz.PromptUnknown, "", fmt.Errorf("ml classify failed: %w", err)
	// }

	// println("gRPC Classify success, type =", resp.Type)

	switch strings.ToLower("Find_Item") {
	case "find_item":
		return biz.PromptFindItem, prompt, nil
	case "select_item":
		return biz.PromptSelectOption, prompt, nil
	case "make_payment":
		return biz.PromptMakePayment, prompt, nil
	default:
		return biz.PromptUnknown, prompt, nil
	}
}

//
// =======================
// PRODUCT SERVICE (gRPC)
// =======================
//

func (r *mainRepo) FindItem(ctx context.Context, query string) (map[string]interface{}, error) {

	// resp, err := r.productClient.SearchProducts(ctx, &productpb.SearchProductsRequest{
	// 	Query: query,
	// })
	// if err != nil {
	// 	return nil, err
	// }

	return map[string]interface{}{
		"products": "Products",
	}, nil
}

func (r *mainRepo) SelectItem(ctx context.Context, id int64) (map[string]interface{}, error) {

	// resp, err := r.productClient.GetProduct(ctx, &productpb.GetProductRequest{
	// 	Id: id,
	// })
	// if err != nil {
	// 	return nil, err
	// }

	return map[string]interface{}{
		"id":    1,
		"name":  "Something",
		"price": 1,
	}, nil
}

//
// =======================
// PAYMENT SERVICE (gRPC)
// =======================
//

func (r *mainRepo) MakePayment(ctx context.Context, method string) (map[string]interface{}, error) {
	// paymentMethod, err := parsePaymentMethod(method)

	// resp, err := r.paymentClient.Pay(ctx, &paymentpb.PayRequest{
	// 	Method: paymentMethod,
	// })
	// if err != nil {
	// 	return nil, err
	// }

	return map[string]interface{}{
		"status": "Success",
	}, nil
}

func parsePaymentMethod(method string) (string, error) {
	method = strings.ToUpper(method)

	// if v, ok := paymentpb.PaymentMethod_value[method]; ok {
	// 	return paymentpb.PaymentMethod(v), nil
	// }

	return "PAYMENT_UNSPECIFIED", nil
}
