package main

import (
	"context"
	"errors"
	"log"

	"matching-service/internal/conf"
	"matching-service/internal/data"
	"matching-service/internal/server"
	"matching-service/pkg/lock"
)

// application 保存服务启动后需要统一管理的组件。
type application struct {
	cfg              *conf.Bootstrap          // cfg 是服务启动配置
	data             *data.Data               // data 是数据层资源
	shardLock        *lock.RedisLock          // shardLock 是撮合分片分布式锁
	rocketMQConsumer *server.RocketMQConsumer // rocketMQConsumer 是 RocketMQ 顺序消费者
	expireWorker     *server.ExpireWorker     // expireWorker 是超时扫描任务
	grpcServer       *server.GRPCServer       // grpcServer 是 gRPC 服务
	metricsServer    *server.MetricsServer    // metricsServer 是 Prometheus 指标服务
}

// Run 启动应用内的后台任务和 gRPC 服务。
func (a *application) Run(ctx context.Context, stop context.CancelFunc) {
	go a.runRocketMQ(ctx, stop)
	go a.expireWorker.Run(ctx)
	go a.runGRPC(ctx, stop)
	go a.runMetrics(ctx, stop)
	log.Printf("matching-service 启动成功，GRPC=%s，Metrics=%s", a.cfg.Server.GRPC.Addr, a.cfg.Server.Metrics.Addr)
}

// Close 关闭应用持有的外部资源。
func (a *application) Close() {
	if a == nil {
		return
	}
	if a.shardLock != nil {
		if err := a.shardLock.Close(); err != nil {
			log.Printf("关闭 Redis 失败: %v", err)
		}
	}
	if a.data != nil {
		if err := a.data.Close(); err != nil {
			log.Printf("关闭数据库失败: %v", err)
		}
	}
}

// runRocketMQ 启动 RocketMQ 消费，异常时触发服务退出。
func (a *application) runRocketMQ(ctx context.Context, stop context.CancelFunc) {
	if err := a.rocketMQConsumer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("RocketMQ 消费停止: %v", err)
		stop()
	}
}

// runGRPC 启动 gRPC 服务，异常时触发服务退出。
func (a *application) runGRPC(ctx context.Context, stop context.CancelFunc) {
	if err := a.grpcServer.Run(ctx); err != nil {
		log.Printf("gRPC 服务停止: %v", err)
		stop()
	}
}

// runMetrics 启动 Prometheus 指标服务，异常时触发服务退出。
func (a *application) runMetrics(ctx context.Context, stop context.CancelFunc) {
	if err := a.metricsServer.Run(ctx); err != nil {
		log.Printf("Prometheus 指标服务停止: %v", err)
		stop()
	}
}
