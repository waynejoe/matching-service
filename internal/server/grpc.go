package server

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport/grpc"

	v1 "matching-service/pkg/api/matching/v1"
	"matching-service/internal/conf"
	appmw "matching-service/internal/middleware"
	"matching-service/internal/service"
	"matching-service/pkg/toolbox/kratox"
)

// NewGRPCServer 根据配置创建带中间件的 gRPC 服务。
func NewGRPCServer(c *conf.Server, matching *service.MatchingService, logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			kratox.SentryMiddleware(),
			appmw.TraceHeaderMiddleware(),
			validate.Validator(),
			logging.Server(logger),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterMatchingServiceServer(srv, matching)
	return srv
}
