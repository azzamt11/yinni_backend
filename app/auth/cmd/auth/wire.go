//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"yinni_backend/app/auth/internal/biz"
	"yinni_backend/app/auth/internal/data"
	"yinni_backend/app/auth/internal/server"
	"yinni_backend/app/auth/internal/service"
	"yinni_backend/internal/conf"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(confServer *conf.Server, confAuth *conf.Auth, confData *conf.Data, logger log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
