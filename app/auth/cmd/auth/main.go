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

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string = "auth-service" // Set default value
	// Version is the version of the compiled software.
	Version string = "v1.0.0" // Set default value
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	// You can override these with build flags if needed
	flag.StringVar(&Name, "name", "auth-service", "service name")
	flag.StringVar(&Version, "version", "v1.0.0", "service version")
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
	)
}

func main() {
	flag.Parse()
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name, // This should now have the value
		"service.version", Version, // This should now have the value
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)

	// Add a log helper
	logHelper := log.NewHelper(logger)
	logHelper.Info("Starting auth service...")

	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
			env.NewSource(
				env.WithPrefix(""), // 🔥 THIS IS THE KEY LINE
			),
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

	// ===== ADD THIS: Log the config values =====
	if bc.Server != nil && bc.Server.Http != nil {
		logHelper.Infof("HTTP port from config: %s", bc.Server.Http.Addr)
	} else {
		logHelper.Error("HTTP config is nil!")
	}

	if bc.Server != nil && bc.Server.Grpc != nil {
		logHelper.Infof("gRPC port from config: %s", bc.Server.Grpc.Addr)
	} else {
		logHelper.Error("gRPC config is nil!")
	}
	// ===== END OF ADDED CODE =====

	app, cleanup, err := wireApp(bc.Server, bc.Auth, bc.Data, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
