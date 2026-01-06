package data

import (
	"context"
	"strings"
	"time"

	"yinni_backend/ent"
	"yinni_backend/ent/migrate"
	"yinni_backend/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewAuthRepo)

// Data .
type Data struct {
	ent *ent.Client
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	logHelper := log.NewHelper(log.With(logger, "module", "auth/data"))

	logHelper.Infof("Connecting to database: %s", c.Database.Source)

	// Simple delay to ensure MySQL is ready
	logHelper.Info("Waiting for MySQL to initialize...")
	time.Sleep(5 * time.Second)

	// Create Ent client
	client, err := ent.Open("mysql", c.Database.Source)
	if err != nil {
		logHelper.Errorf("Failed to open database: %v", err)
		return nil, nil, err
	}

	// Run migration with a simple retry
	logHelper.Info("Running database migration...")
	ctx := context.Background()

	var migrationErr error
	for i := 0; i < 5; i++ {
		migrationErr = client.Schema.Create(
			ctx,
			migrate.WithDropIndex(false),
			migrate.WithDropColumn(false),
		)
		if migrationErr == nil {
			logHelper.Info("Database schema created/updated successfully")
			break
		}

		logHelper.Warnf("Migration attempt %d failed: %v", i+1, migrationErr)

		// If it's a connection error, wait and retry
		if strings.Contains(migrationErr.Error(), "connection refused") ||
			strings.Contains(migrationErr.Error(), "dial tcp") {
			logHelper.Info("Waiting 3 seconds before retry...")
			time.Sleep(3 * time.Second)
			continue
		}

		// For other errors, break
		break
	}

	if migrationErr != nil {
		logHelper.Errorf("Failed to create schema: %v", migrationErr)
		// Don't fail immediately - maybe the table already exists
		logHelper.Warn("Schema creation failed, but continuing anyway...")
	}

	cleanup := func() {
		logHelper.Info("closing the data resources")
		client.Close()
	}

	logHelper.Info("Database connection established successfully")
	return &Data{ent: client}, cleanup, nil
}
