package server

import (
	v1 "yinni_backend/api/product/v1"
	"yinni_backend/app/product/internal/service"
	"yinni_backend/internal/conf"
	"yinni_backend/pkg/middleware"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/rs/cors"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, authConf *conf.Auth, product *service.ProductService, logger log.Logger) *http.Server {
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
			middleware.JWT(authConf.JwtSecret),
		),
		http.Filter(corsHandler.Handler),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}

	srv := http.NewServer(opts...)
	v1.RegisterProductHTTPServer(srv, product)
	return srv
}
