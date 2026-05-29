//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	"matching-service/internal/biz"
	"matching-service/internal/conf"
	"matching-service/internal/data"
	"matching-service/internal/server"
	"matching-service/internal/service"
	"matching-service/pkg/toolbox/redisx"
)

// wireApp 使用 Wire 装配 Kratos 应用。
func wireApp(*conf.Bootstrap, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(
		provideContext,
		wire.FieldsOf(new(*conf.Bootstrap), "Server"),
		data.ProviderSet,
		redisx.ProviderSet,
		biz.ProviderSet,
		service.ProviderSet,
		server.ProviderSet,
		newApp,
	))
}

func provideContext() context.Context {
	return context.Background()
}
