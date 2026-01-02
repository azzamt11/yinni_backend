package data

import (
	"context"
	"yinni_backend/ent"
	_ "yinni_backend/ent/runtime"
	"yinni_backend/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewProductRepo)

// Data .
type Data struct {
	ent *ent.Client
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	log := log.NewHelper(logger)

	client, err := ent.Open(
		"mysql",
		c.Database.Source,
	)

	if err != nil {
		return nil, nil, err
	}

	if err := client.Schema.Create(context.Background()); err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		log.Info("closing the data resources")
	}
	return &Data{ent: client}, cleanup, nil
}
