package server

import (
	"context"
	"time"

	"matching-service/internal/conf"
	"matching-service/internal/data"
	"matching-service/internal/service"
	"matching-service/pkg/toolbox/redisx"
)

// HealthChecker 负责检查服务依赖健康状态。
type HealthChecker struct {
	cfg       *conf.Bootstrap // cfg 是服务配置
	data      *data.Data      // data 是数据库资源
	shardLock *redisx.Lock    // shardLock 是 Redis 分片锁
}

// NewHealthChecker 创建健康检查器。
func NewHealthChecker(cfg *conf.Bootstrap, data *data.Data, shardLock *redisx.Lock) *HealthChecker {
	return &HealthChecker{cfg: cfg, data: data, shardLock: shardLock}
}

// Check 检查 MySQL、Redis 和 RocketMQ。
func (c *HealthChecker) Check(ctx context.Context) service.HealthResult {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	components := []service.HealthComponent{
		c.checkMySQL(ctx),
		c.checkRedis(ctx),
		c.checkRocketMQ(ctx),
	}
	ok := true
	for _, item := range components {
		if !item.OK {
			ok = false
			break
		}
	}
	return service.HealthResult{OK: ok, Components: components}
}

// checkMySQL 检查 MySQL。
func (c *HealthChecker) checkMySQL(ctx context.Context) service.HealthComponent {
	if err := c.data.Ping(ctx); err != nil {
		return service.HealthComponent{Name: "mysql", OK: false, Message: err.Error()}
	}
	return service.HealthComponent{Name: "mysql", OK: true, Message: "ok"}
}

// checkRedis 检查 Redis。
func (c *HealthChecker) checkRedis(ctx context.Context) service.HealthComponent {
	if err := c.shardLock.Ping(ctx); err != nil {
		return service.HealthComponent{Name: "redis", OK: false, Message: err.Error()}
	}
	return service.HealthComponent{Name: "redis", OK: true, Message: "ok"}
}

// checkRocketMQ 检查 RocketMQ producer 是否可启动。
func (c *HealthChecker) checkRocketMQ(ctx context.Context) service.HealthComponent {
	producer, err := NewRocketMQProducer(c.cfg.Data.Rocketmq)
	if err != nil {
		return service.HealthComponent{Name: "rocketmq", OK: false, Message: err.Error()}
	}
	if err := producer.Start(); err != nil {
		return service.HealthComponent{Name: "rocketmq", OK: false, Message: err.Error()}
	}
	done := make(chan error, 1)
	go func() {
		done <- producer.Shutdown()
	}()
	select {
	case err := <-done:
		if err != nil {
			return service.HealthComponent{Name: "rocketmq", OK: false, Message: err.Error()}
		}
	case <-ctx.Done():
		return service.HealthComponent{Name: "rocketmq", OK: false, Message: ctx.Err().Error()}
	}
	return service.HealthComponent{Name: "rocketmq", OK: true, Message: "ok"}
}
