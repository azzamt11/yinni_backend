package data

import (
	"yinni_backend/internal/conf"

	classifierpb "yinni_backend/api/classifier/v1"
	paymentpb "yinni_backend/api/payment/v1"
	productpb "yinni_backend/api/product/v1"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"google.golang.org/grpc"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewProductClient, NewPaymentClient, NewClassifierClient, NewPromptRepo)

// Data .
type Data struct {
	// TODO wrapped database client
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	return &Data{}, cleanup, nil
}

func NewProductClient(
	c *conf.Services,
) (productpb.ProductClient, error) {

	conn, err := grpc.Dial(
		c.ProductServiceEndpoint,
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	return productpb.NewProductClient(conn), nil
}

func NewPaymentClient(
	c *conf.Services,
) (paymentpb.PaymentClient, error) {

	conn, err := grpc.Dial(
		c.PaymentServiceEndpoint,
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	return paymentpb.NewPaymentClient(conn), nil
}

func NewClassifierClient(
	c *conf.Services,
) (classifierpb.ClassifierClient, error) {

	conn, err := grpc.Dial(
		c.ClassifierServiceEndpoint,
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	return classifierpb.NewClassifierClient(conn), nil
}
