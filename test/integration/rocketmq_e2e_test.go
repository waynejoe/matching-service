//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"matching-service/internal/conf"
	"matching-service/pkg/model"
	"matching-service/internal/server"
	"matching-service/internal/service"
	"matching-service/pkg/toolbox/mqx"
)

// TestRocketMQE2EPartialMatchSucceeded 验证 RocketMQ 真实收发链路可以完成少发成交。
func TestRocketMQE2EPartialMatchSucceeded(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv(t, ctx)
	defer env.close()

	prefix := fmt.Sprintf("IT_RMQ_E2E_%d", time.Now().UnixNano())
	env.cleanup(t, prefix)
	defer env.cleanup(t, prefix)
	if env.cfg.Data.Rocketmq.Consumer == nil {
		env.cfg.Data.Rocketmq.Consumer = &conf.Consumer{}
	}
	env.cfg.Data.Rocketmq.Consumer.Group = prefix

	matchingConsumer := service.NewMatchingConsumer(env.uc, env.m)
	consumerMgr, err := server.NewConsumerManager(env.cfg, matchingConsumer, log.DefaultLogger)
	if err != nil {
		t.Fatalf("创建 RocketMQ consumer 失败: %v", err)
	}
	consumerCtx, stopConsumer := context.WithCancel(ctx)
	consumerErr := startMQConsumerManager(t, consumerCtx, consumerMgr)
	defer stopMQConsumerManager(t, stopConsumer, consumerErr)

	producer, err := server.NewRocketMQProducer(env.cfg.Data.Rocketmq)
	if err != nil {
		t.Fatalf("创建 RocketMQ producer 失败: %v", err)
	}
	if err := producer.Start(); err != nil {
		t.Fatalf("启动 RocketMQ producer 失败: %v", err)
	}
	defer func() {
		if err := producer.Shutdown(); err != nil {
			t.Fatalf("关闭 RocketMQ producer 失败: %v", err)
		}
	}()

	basketNo := prefix + "_B"
	withdrawNo := prefix + "_W"
	depositNo := prefix + "_D"
	channel := fmt.Sprintf("IT%d", time.Now().UnixNano()%1000000000)
	currency := "TST"
	expireAt := time.Now().Add(time.Hour)
	sendCtx, cancelSend := context.WithTimeout(ctx, 10*time.Second)
	defer cancelSend()
	if err := producer.SendWithdraw(sendCtx, prefix+"_WE", model.WithdrawBasket{
		BasketNo:     basketNo,
		WithdrawNo:   withdrawNo,
		Channel:      channel,
		Currency:     currency,
		TargetAmount: 1000,
		ExpireAt:     expireAt,
	}, 0); err != nil {
		t.Fatalf("发送出金消息失败: %v", err)
	}
	waitFor(t, 15*time.Second, func() bool {
		var count int64
		env.data.DB.WithContext(ctx).Model(&model.WithdrawBasket{}).Where("basket_no = ?", basketNo).Count(&count)
		return count == 1
	})

	if err := producer.SendDeposit(sendCtx, prefix+"_DE", model.DepositOrder{
		DepositNo: depositNo,
		Channel:   channel,
		Currency:  currency,
		Amount:    300,
		ExpireAt:  expireAt.Add(time.Hour),
	}, 0); err != nil {
		t.Fatalf("发送入金消息失败: %v", err)
	}
	waitFor(t, 15*time.Second, func() bool {
		var basket model.WithdrawBasket
		if err := env.data.DB.WithContext(ctx).Where("basket_no = ?", basketNo).First(&basket).Error; err != nil {
			return false
		}
		return basket.CurrentAmount == 300 && basket.Status == model.StatusWaiting
	})

	if _, err := env.uc.ExpireTimeouts(ctx, expireAt.Add(time.Second), 100); err != nil {
		t.Fatalf("触发超时失败: %v", err)
	}
	var record model.MatchRecord
	if err := env.data.DB.WithContext(ctx).Where("basket_no = ?", basketNo).First(&record).Error; err != nil {
		t.Fatalf("查询撮合记录失败: %v", err)
	}
	if record.MatchedAmount != 300 || record.ShortAmount != 700 {
		t.Fatalf("期望成交 300 少发 700，实际成交 %d 少发 %d", record.MatchedAmount, record.ShortAmount)
	}
	assertEventStatus(t, env, withdrawNo, model.EventStatusSucceeded)
	assertEventStatus(t, env, depositNo, model.EventStatusSucceeded)
}

func startMQConsumerManager(t *testing.T, ctx context.Context, mgr *mqx.ConsumerManager) <-chan error {
	t.Helper()
	errCh := make(chan error, 1)
	go func() {
		errCh <- mgr.Start(ctx)
	}()
	select {
	case err := <-errCh:
		t.Fatalf("启动 RocketMQ consumer 失败: %v", err)
	case <-time.After(3 * time.Second):
	}
	return errCh
}

func stopMQConsumerManager(t *testing.T, cancel context.CancelFunc, errCh <-chan error) {
	t.Helper()
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("关闭 RocketMQ consumer 失败: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("关闭 RocketMQ consumer 超时")
	}
}

// waitFor 等待条件在超时时间内变成 true。
func waitFor(t *testing.T, timeout time.Duration, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("等待条件超时: %s", timeout)
}
