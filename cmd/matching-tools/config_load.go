package main

import (
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"

	"matching-service/internal/conf"
)

func loadBootstrap(path string) (*conf.Bootstrap, error) {
	c := config.New(config.WithSource(file.NewSource(path)))
	defer func() { _ = c.Close() }()
	if err := c.Load(); err != nil {
		return nil, err
	}
	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		return nil, err
	}
	return &bc, nil
}
