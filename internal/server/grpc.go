package server

import (
	"context"
	"errors"
	"log"
	"net"

	"google.golang.org/grpc"

	"matching-service/internal/conf"
	"matching-service/internal/service"
	v1 "matching-service/api/matching/v1"
)

// GRPCServer 是 gRPC 服务封装。
type GRPCServer struct {
	server *grpc.Server // server 是 gRPC 服务实例
	addr   string       // addr 是监听地址
}

// NewGRPCServer 根据配置创建 gRPC 服务。
func NewGRPCServer(cfg *conf.Bootstrap, matching *service.MatchingService) *GRPCServer {
	srv := grpc.NewServer()
	v1.RegisterMatchingServiceServer(srv, matching)
	return &GRPCServer{
		server: srv,
		addr:   cfg.Server.GRPC.Addr,
	}
}

// Run 启动 gRPC 服务。
func (s *GRPCServer) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		s.server.GracefulStop()
	}()
	log.Printf("gRPC 服务监听: %s", s.addr)
	if err := s.server.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return err
	}
	return nil
}
