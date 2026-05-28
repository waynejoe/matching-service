package server

import (
	"context"
	"log"
	"time"

	"matching-service/internal/biz"
	"matching-service/internal/conf"
)

// ExpireWorker 定时处理撮合超时数据。
type ExpireWorker struct {
	uc       *biz.MatchingUsecase // uc 是撮合业务用例
	interval time.Duration        // interval 是扫描间隔
	limit    int                  // limit 是每轮扫描数量
}

// NewExpireWorker 根据配置创建超时处理 worker。
func NewExpireWorker(cfg *conf.Bootstrap, uc *biz.MatchingUsecase) *ExpireWorker {
	return &ExpireWorker{
		uc:       uc,
		interval: time.Duration(cfg.Match.ExpireIntervalSec) * time.Second,
		limit:    cfg.Match.ExpireBatchSize,
	}
}

// Run 启动超时扫描循环。
func (w *ExpireWorker) Run(ctx context.Context) {
	if w.interval <= 0 {
		w.interval = 5 * time.Second
	}
	if w.limit <= 0 {
		w.limit = 500
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	w.scan(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.scan(ctx)
		}
	}
}

// scan 执行一轮超时扫描。
func (w *ExpireWorker) scan(ctx context.Context) {
	result, err := w.uc.ExpireTimeouts(ctx, time.Now(), w.limit)
	if err != nil {
		log.Printf("处理超时数据失败: %v", err)
		return
	}
	if result.WaitingDeposits > 0 || result.LockedDeposits > 0 || result.Baskets > 0 {
		log.Printf("处理超时数据完成，待撮合入金=%d，已挂入入金=%d，出金篮子=%d", result.WaitingDeposits, result.LockedDeposits, result.Baskets)
	}
}
