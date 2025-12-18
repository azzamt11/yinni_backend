package data

import (
	"context"
	"yinni_backend/ent" // Import your generated ent code
	"yinni_backend/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	// Import the database driver
	_ "github.com/go-sql-driver/mysql"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewGreeterRepo)

// Data .
type Data struct {
	db *ent.Client // Add this field
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	helper := log.NewHelper(logger)

	// 1. Initialize Ent Client
	client, err := ent.Open(
		c.Database.Driver,
		c.Database.Source,
	)
	if err != nil {
		helper.Errorf("failed opening connection to db: %v", err)
		return nil, nil, err
	}

	// 2. Run the auto migration (optional but recommended during development)
	if err := client.Schema.Create(context.Background()); err != nil {
		helper.Errorf("failed creating schema resources: %v", err)
		return nil, nil, err
	}

	d := &Data{
		db: client,
	}

	// 3. Update cleanup to close the database connection
	cleanup := func() {
		helper.Info("closing the data resources")
		if err := d.db.Close(); err != nil {
			helper.Error(err)
		}
	}

	return d, cleanup, nil
}
