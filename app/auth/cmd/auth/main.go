package main

import (
	"flag"
	"os"

	"yinni_backend/internal/conf"

	_ "github.com/go-sql-driver/mysql"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/env"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "go.uber.org/automaxprocs"
)

var (
	Name     = "auth-service"
	Version  = "v1.0.0"
	flagconf string
	id, _    = os.Hostname()
)

func init() {
	flag.StringVar(&Name, "name", Name, "service name")
	flag.StringVar(&Version, "version", Version, "service version")
	flag.StringVar(&flagconf, "conf", "/data/conf", "config path")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Logger(logger),
		kratos.Server(gs, hs),
	)
}

func main() {
	flag.Parse()

	logger := log.With(
		log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)

	logHelper := log.NewHelper(logger)
	logHelper.Info("Starting service...")

	// 🔴 IMPORTANT: env.NewSource() ONLY
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
			env.NewSource(),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	// ---- sanity logs ----
	logHelper.Infof("HTTP addr: %s", bc.Server.Http.Addr)
	logHelper.Infof("gRPC addr: %s", bc.Server.Grpc.Addr)
	logHelper.Infof("DB source: %s", bc.Data.Database.Source)
	logHelper.Infof("JWT secret: %s", bc.Auth.JwtSecret)
	logHelper.Infof("JWT expire: %s", bc.Auth.JwtExpire)

	app, cleanup, err := wireApp(bc.Server, bc.Auth, bc.Data, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	if err := app.Run(); err != nil {
		panic(err)
	}
}
