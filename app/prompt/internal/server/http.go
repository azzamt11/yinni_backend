package server

import (
	"context"
	v1 "yinni_backend/api/prompt/v1"
	"yinni_backend/app/prompt/internal/service"
	"yinni_backend/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/rs/cors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, authConf *conf.Auth, prompt *service.PromptService, logger log.Logger) *http.Server {
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(
				recovery.WithHandler(func(ctx context.Context, req, err interface{}) error {
					log.Error("PANIC RECOVERED:", err)
					return status.Errorf(codes.Internal, "internal server error")
				}),
			),
			logging.Server(logger),
			//middleware.JWT(authConf.JwtSecret),
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

	log.NewHelper(logger).Info("HTTP server initialized, registering Prompt handlers")
	v1.RegisterPromptHTTPServer(srv, prompt)
	return srv
}
