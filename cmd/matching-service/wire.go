//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"

	"matching-service/internal/biz"
	"matching-service/internal/conf"
	"matching-service/internal/data"
	"matching-service/internal/server"
	"matching-service/internal/service"
	"matching-service/pkg/lock"
)

// newApplication 使用 Wire 装配服务依赖。
func newApplication(ctx context.Context, confPath string) (*application, error) {
	wire.Build(
		conf.Load,
		data.ProviderSet,
		lock.ProviderSet,
		biz.ProviderSet,
		service.ProviderSet,
		server.ProviderSet,
		wire.Struct(new(application), "*"),
	)
	return nil, nil
}
