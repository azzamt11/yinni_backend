package data

import (
	"yinni_backend/ent"
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

	// Create Ent client
	client, err := ent.Open("mysql", c.Database.Source)
	if err != nil {
		logHelper.Errorf("Failed to open database: %v", err)
		return nil, nil, err
	}

	// DO NOT create schema here!
	// The tables should already exist or will be created by individual services

	cleanup := func() {
		logHelper.Info("closing the data resources")
		client.Close()
	}

	logHelper.Info("Database connection established successfully")
	return &Data{ent: client}, cleanup, nil
}
