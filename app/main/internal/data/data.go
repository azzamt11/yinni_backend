package data

import (
	"yinni_backend/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewMainRepo)

// Data .
type Data struct {
	// TODO wrapped database client
}

// NewData .
func NewData(c *conf.Data) (*Data, func(), error) {
	cleanup := func() {
		log.Info("closing the data resources")
	}
	return &Data{}, cleanup, nil
}

// func NewProductClient(
// 	c *conf.Services,
// ) (productpb.ProductClient, error) {

// 	conn, err := grpc.Dial(
// 		c.ProductServiceEndpoint,
// 		grpc.WithInsecure(), // internal network
// 		grpc.WithBlock(),
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return productpb.NewProductClient(conn), nil
// }

// func NewPaymentClient(
// 	c *conf.Services,
// ) (paymentpb.PaymentClient, error) {

// 	conn, err := grpc.Dial(
// 		c.PaymentServiceEndpoint,
// 		grpc.WithInsecure(),
// 		grpc.WithBlock(),
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return paymentpb.NewPaymentClient(conn), nil
// }

// func NewClassifierClient(
// 	c *conf.Services,
// ) (classifierpb.ClassifierClient, error) {

// 	conn, err := grpc.Dial(
// 		c.ClassifierServiceEndpoint,
// 		grpc.WithInsecure(),
// 		grpc.WithBlock(),
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return classifierpb.NewClassifierClient(conn), nil
// }
