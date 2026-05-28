package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"matching-service/internal/biz"
	"matching-service/internal/conf"
)

// MetricsServer 是 Prometheus 指标服务。
type MetricsServer struct {
	addr   string       // addr 是指标监听地址
	server *http.Server // server 是 HTTP 服务实例
}

// NewMetricsServer 创建 Prometheus 指标服务。
func NewMetricsServer(cfg *conf.Bootstrap, metrics *biz.Metrics) *MetricsServer {
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	registerMatchingCounters(registry, metrics)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	return &MetricsServer{
		addr: cfg.Server.Metrics.Addr,
		server: &http.Server{
			Addr:              cfg.Server.Metrics.Addr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

// Run 启动 Prometheus 指标服务。
func (s *MetricsServer) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Prometheus 指标服务关闭失败: %v", err)
		}
	}()
	log.Printf("Prometheus 指标监听: %s", s.addr)
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// registerMatchingCounters 注册撮合服务指标。
func registerMatchingCounters(registry *prometheus.Registry, metrics *biz.Metrics) {
	registerCounter(registry, "matching_deposit_consume_success_total", "入金消息消费成功次数。", func() float64 {
		return float64(metrics.Snapshot().DepositConsumeSuccess)
	})
	registerCounter(registry, "matching_deposit_consume_failed_total", "入金消息消费失败次数。", func() float64 {
		return float64(metrics.Snapshot().DepositConsumeFailed)
	})
	registerCounter(registry, "matching_withdraw_consume_success_total", "出金消息消费成功次数。", func() float64 {
		return float64(metrics.Snapshot().WithdrawConsumeSuccess)
	})
	registerCounter(registry, "matching_withdraw_consume_failed_total", "出金消息消费失败次数。", func() float64 {
		return float64(metrics.Snapshot().WithdrawConsumeFailed)
	})
	registerCounter(registry, "matching_success_total", "足额撮合成功次数。", func() float64 {
		return float64(metrics.Snapshot().MatchSuccess)
	})
	registerCounter(registry, "matching_short_match_success_total", "少发撮合成功次数。", func() float64 {
		return float64(metrics.Snapshot().ShortMatchSuccess)
	})
	registerCounter(registry, "matching_expired_waiting_deposits_total", "等待中入金单过期次数。", func() float64 {
		return float64(metrics.Snapshot().ExpiredWaitingDeposits)
	})
	registerCounter(registry, "matching_expired_locked_deposits_total", "锁定中入金单过期次数。", func() float64 {
		return float64(metrics.Snapshot().ExpiredLockedDeposits)
	})
	registerCounter(registry, "matching_expired_baskets_total", "出金篮子过期次数。", func() float64 {
		return float64(metrics.Snapshot().ExpiredBaskets)
	})
	registerCounter(registry, "matching_event_failed_total", "事件处理失败次数。", func() float64 {
		return float64(metrics.Snapshot().EventFailed)
	})
	registerCounter(registry, "matching_event_retried_total", "事件重试次数。", func() float64 {
		return float64(metrics.Snapshot().EventRetried)
	})
}

// registerCounter 注册单个 Prometheus counter。
func registerCounter(registry *prometheus.Registry, name string, help string, value func() float64) {
	registry.MustRegister(prometheus.NewCounterFunc(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, value))
}
