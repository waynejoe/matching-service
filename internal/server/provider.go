package server

import (
	"github.com/google/wire"

	"matching-service/internal/service"
)

// ProviderSet 是服务监听层 Wire provider 集合。
var ProviderSet = wire.NewSet(
	NewGRPCServer,
	NewHealthChecker,
	NewMetricsServer,
	NewConsumerManager,
	NewMQProducer,
	wire.Bind(new(service.HealthChecker), new(*HealthChecker)),
)
