//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"matching-service/internal/model"
	"matching-service/internal/service"
)

// TestWorkerMessagesPartialMatchSucceeded 验证 RocketMQ 消息入口可以完成少发成交。
func TestWorkerMessagesPartialMatchSucceeded(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv(t, ctx)
	defer env.close()
	prefix := fmt.Sprintf("IT_WORKER_%d", time.Now().UnixNano())
	env.cleanup(t, prefix)
	defer env.cleanup(t, prefix)

	matchingConsumer := service.NewMatchingConsumer(env.uc, env.m)
	expireAt := time.Now().Add(time.Hour)
	withdrawBody := mustJSON(t, service.WithdrawEventMessage{
		EventID: prefix + "_WE",
		Topic:   env.cfg.Data.RocketMQ.WithdrawTopic,
		Data: model.WithdrawBasket{
			BasketNo:     prefix + "_B",
			WithdrawNo:   prefix + "_W",
			Channel:      "IT_WORKER",
			Currency:     "TST",
			TargetAmount: 1000,
			ExpireAt:     expireAt,
		},
	})
	if err := matchingConsumer.HandleWithdrawMessage(ctx, withdrawBody); err != nil {
		t.Fatalf("处理出金消息失败: %v", err)
	}

	depositBody := mustJSON(t, service.DepositEventMessage{
		EventID: prefix + "_DE",
		Topic:   env.cfg.Data.RocketMQ.DepositTopic,
		Data: model.DepositOrder{
			DepositNo: prefix + "_D",
			Channel:   "IT_WORKER",
			Currency:  "TST",
			Amount:    300,
			ExpireAt:  expireAt.Add(time.Hour),
		},
	})
	if err := matchingConsumer.HandleDepositMessage(ctx, depositBody); err != nil {
		t.Fatalf("处理入金消息失败: %v", err)
	}
	if _, err := env.uc.ExpireTimeouts(ctx, expireAt.Add(time.Second), 100); err != nil {
		t.Fatalf("触发超时失败: %v", err)
	}

	var record model.MatchRecord
	if err := env.data.DB.WithContext(ctx).Where("basket_no = ?", prefix+"_B").First(&record).Error; err != nil {
		t.Fatalf("查询撮合记录失败: %v", err)
	}
	if record.MatchedAmount != 300 || record.ShortAmount != 700 {
		t.Fatalf("期望成交 300 少发 700，实际成交 %d 少发 %d", record.MatchedAmount, record.ShortAmount)
	}
	assertEventStatus(t, env, prefix+"_W", model.EventStatusSucceeded)
	assertEventStatus(t, env, prefix+"_D", model.EventStatusSucceeded)
}

// TestWorkerMessageFailedThenRetried 验证失败事件可以重新消费并成功。
func TestWorkerMessageFailedThenRetried(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv(t, ctx)
	defer env.close()
	prefix := fmt.Sprintf("IT_WORKER_RETRY_%d", time.Now().UnixNano())
	env.cleanup(t, prefix)
	defer env.cleanup(t, prefix)

	matchingConsumer := service.NewMatchingConsumer(env.uc, env.m)
	expiredBody := mustJSON(t, service.WithdrawEventMessage{
		EventID: prefix + "_WE",
		Topic:   env.cfg.Data.RocketMQ.WithdrawTopic,
		Data: model.WithdrawBasket{
			BasketNo:     prefix + "_B",
			WithdrawNo:   prefix + "_W",
			Channel:      "IT_WORKER_RETRY",
			Currency:     "TST",
			TargetAmount: 1000,
			ExpireAt:     time.Now().Add(-time.Minute),
		},
	})
	if err := matchingConsumer.HandleWithdrawMessage(ctx, expiredBody); err == nil {
		t.Fatalf("过期出金消息期望失败")
	}
	assertEventStatus(t, env, prefix+"_W", model.EventStatusFailed)

	validBody := mustJSON(t, service.WithdrawEventMessage{
		EventID: prefix + "_WE",
		Topic:   env.cfg.Data.RocketMQ.WithdrawTopic,
		Data: model.WithdrawBasket{
			BasketNo:     prefix + "_B",
			WithdrawNo:   prefix + "_W",
			Channel:      "IT_WORKER_RETRY",
			Currency:     "TST",
			TargetAmount: 1000,
			ExpireAt:     time.Now().Add(time.Hour),
		},
	})
	if err := matchingConsumer.HandleWithdrawMessage(ctx, validBody); err != nil {
		t.Fatalf("重试出金消息失败: %v", err)
	}
	assertEventStatus(t, env, prefix+"_W", model.EventStatusSucceeded)
	assertBasketStatus(t, env, prefix+"_B", model.StatusWaiting)

	snapshot := env.m.Snapshot()
	if snapshot.WithdrawConsumeFailed != 1 || snapshot.WithdrawConsumeSuccess != 1 {
		t.Fatalf("消费指标不正确: failed=%d success=%d", snapshot.WithdrawConsumeFailed, snapshot.WithdrawConsumeSuccess)
	}
	if snapshot.EventFailed != 1 || snapshot.EventRetried != 1 {
		t.Fatalf("事件指标不正确: failed=%d retried=%d", snapshot.EventFailed, snapshot.EventRetried)
	}
}

// mustJSON 把消息编码成 JSON。
func mustJSON(t *testing.T, in any) []byte {
	t.Helper()
	out, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("编码 JSON 失败: %v", err)
	}
	return out
}

// assertEventStatus 断言消费幂等事件状态。
func assertEventStatus(t *testing.T, env *testEnv, bizNo string, status int32) {
	t.Helper()
	var event model.EventInbox
	if err := env.data.DB.Where("biz_no = ?", bizNo).First(&event).Error; err != nil {
		t.Fatalf("查询幂等事件失败: %v", err)
	}
	if event.Status != status {
		t.Fatalf("事件 %s 期望状态 %d，实际 %d", bizNo, status, event.Status)
	}
}
