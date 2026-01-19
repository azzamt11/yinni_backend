package server

import (
	nethttp "net/http"

	v1 "yinni_backend/api/payment/v1"
	"yinni_backend/app/payment/internal/service"
	"yinni_backend/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

func NewHTTPServer(
	c *conf.Server,
	payment *service.PaymentService,
	logger log.Logger,
) *khttp.Server {

	opts := []khttp.ServerOption{
		khttp.Middleware(
			recovery.Recovery(),
		),
	}

	if c.Http.Network != "" {
		opts = append(opts, khttp.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, khttp.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, khttp.Timeout(c.Http.Timeout.AsDuration()))
	}

	srv := khttp.NewServer(opts...)

	// Register Kratos HTTP (gRPC-Gateway)
	v1.RegisterPaymentHTTPServer(srv, payment)

	// ✅ Static file server
	// URL:  http://host:port/assets/foo.png
	// Path: app/payment/assets/foo.png
	fileServer := nethttp.StripPrefix(
		"/assets/",
		nethttp.FileServer(nethttp.Dir("/app/assets")),
	)

	srv.HandlePrefix("/assets/", fileServer)

	return srv
}
