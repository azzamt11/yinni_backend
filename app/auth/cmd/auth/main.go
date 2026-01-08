package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"yinni_backend/ent"
	"yinni_backend/ent/user"
	"yinni_backend/internal/conf"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/env"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"golang.org/x/crypto/bcrypt"

	_ "yinni_backend/ent/runtime"

	_ "github.com/go-sql-driver/mysql"
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

	// Load configuration
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

	// Log admin config
	if bc.Admin != nil {
		logHelper.Infof("Admin email: %s", bc.Admin.Email)
		logHelper.Infof("Admin name: %s", bc.Admin.Name)

		// Security warnings
		if bc.Admin.Email == "noemail@yinni.com" {
			logHelper.Warn("⚠️  Using default admin email. Set KRATOS_ADMIN_EMAIL environment variable.")
		}
		if bc.Admin.Password == "admin123" {
			logHelper.Error("🚨 USING WEAK DEFAULT ADMIN PASSWORD! Set KRATOS_ADMIN_PASSWORD immediately!")
		}
	}
	// ----------------------

	app, cleanup, err := wireApp(bc.Server, bc.Auth, bc.Data, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// **NEW: Seed admin AFTER wireApp (tables are created by Ent migrations in wireApp)**
	if bc.Admin != nil && bc.Admin.Email != "noemail@yinni.com" {
		logHelper.Info("Seeding admin user...")

		// Give a moment for migrations to complete
		time.Sleep(2 * time.Second)

		// Use the client from wireApp or create a new one
		go func() {
			// Wait a bit more to ensure migrations are done
			time.Sleep(5 * time.Second)

			if err := seedAdminUser(bc.Admin, bc.Data, logger); err != nil {
				logHelper.Errorf("Failed to seed admin user: %v", err)
			}
		}()
	}

	if err := app.Run(); err != nil {
		panic(err)
	}
}

// hashPassword generates bcrypt hash of the password
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// seedAdminUser creates or updates the admin user from config
func seedAdminUser(adminConf *conf.Admin, dataConf *conf.Data, logger log.Logger) error {
	logHelper := log.NewHelper(log.With(logger, "module", "admin-seed"))

	// Validate admin config
	if adminConf.Email == "" || adminConf.Password == "" || adminConf.Name == "" {
		return fmt.Errorf("admin email, password, and name are required")
	}

	// Connect to database
	drv, err := sql.Open(dialect.MySQL, dataConf.Database.Source)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Get the underlying sql.DB
	db := drv.DB()
	client := ent.NewClient(ent.Driver(drv))
	defer client.Close()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Wait for database to be ready (retry logic)
	logHelper.Info("Waiting for database connection...")
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		if err := db.PingContext(ctx); err == nil {
			logHelper.Info("Database connection established")
			break
		}

		if i == maxRetries-1 {
			return fmt.Errorf("database not ready after %d attempts", maxRetries)
		}

		logHelper.Infof("Database not ready (attempt %d/%d), retrying...", i+1, maxRetries)
		time.Sleep(3 * time.Second)
	}

	// Check if admin user already exists
	adminExists, err := client.User.Query().
		Where(user.EmailEQ(adminConf.Email)).
		Exist(ctx)
	if err != nil {
		// If table doesn't exist yet, that's OK - we'll create it later
		logHelper.Warnf("Could not check admin existence (table may not exist yet): %v", err)
		return nil
	}

	// Hash the password (always hash even if user exists, in case password changed)
	hashedPassword, err := hashPassword(adminConf.Password)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	if adminExists {
		logHelper.Info("Admin user already exists, updating...")

		// Update existing admin user
		updated, err := client.User.Update().
			Where(user.EmailEQ(adminConf.Email)).
			SetPassword(hashedPassword).
			SetName(adminConf.Name).
			SetIsAdmin(true).
			SetRoles([]string{"user", "admin"}).
			Save(ctx)

		if err != nil {
			return fmt.Errorf("failed to update admin user: %w", err)
		}

		logHelper.Infof("Admin user updated successfully (ID: %d)", updated)
		return nil
	}

	logHelper.Info("Creating new admin user...")

	// Create admin user
	created, err := client.User.Create().
		SetName(adminConf.Name).
		SetEmail(adminConf.Email).
		SetPassword(hashedPassword).
		SetIsAdmin(true).
		SetRoles([]string{"user", "admin"}).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	logHelper.Infof("✅ Admin user created successfully (ID: %d, Email: %s)", created.ID, created.Email)
	logHelper.Warn("⚠️  Admin credentials loaded from config. Ensure they are secure in production!")

	return nil
}
