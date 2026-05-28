package service

import (
	"context"
	"encoding/json"

	"matching-service/internal/biz"
	"matching-service/internal/model"
)

// MatchingConsumer 是 RocketMQ 消息处理入口。
type MatchingConsumer struct {
	uc      *biz.MatchingUsecase // uc 是撮合业务用例
	metrics *biz.Metrics         // metrics 是运行指标
}

// DepositEventMessage 是入金 RocketMQ 消息。
type DepositEventMessage struct {
	EventID string             `json:"eventId"` // EventID 是事件唯一 ID
	Topic   string             `json:"topic"`   // Topic 是 RocketMQ 主题
	Data    model.DepositOrder `json:"data"`    // Data 是入金单数据
}

// WithdrawEventMessage 是出金 RocketMQ 消息。
type WithdrawEventMessage struct {
	EventID string               `json:"eventId"` // EventID 是事件唯一 ID
	Topic   string               `json:"topic"`   // Topic 是 RocketMQ 主题
	Data    model.WithdrawBasket `json:"data"`    // Data 是出金篮子数据
}

// NewMatchingConsumer 创建撮合消息处理入口。
func NewMatchingConsumer(uc *biz.MatchingUsecase, metric *biz.Metrics) *MatchingConsumer {
	return &MatchingConsumer{uc: uc, metrics: metric}
}

// HandleDepositMessage 处理入金消息。
func (c *MatchingConsumer) HandleDepositMessage(ctx context.Context, body []byte) error {
	var msg DepositEventMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		c.metrics.IncDepositConsumeFailed()
		return err
	}
	if _, _, err := c.uc.ProcessDepositEvent(ctx, msg.EventID, msg.Topic, &msg.Data); err != nil {
		c.metrics.IncDepositConsumeFailed()
		return err
	}
	c.metrics.IncDepositConsumeSuccess()
	return nil
}

// HandleWithdrawMessage 处理出金消息。
func (c *MatchingConsumer) HandleWithdrawMessage(ctx context.Context, body []byte) error {
	var msg WithdrawEventMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		c.metrics.IncWithdrawConsumeFailed()
		return err
	}
	if _, err := c.uc.ProcessWithdrawEvent(ctx, msg.EventID, msg.Topic, &msg.Data); err != nil {
		c.metrics.IncWithdrawConsumeFailed()
		return err
	}
	c.metrics.IncWithdrawConsumeSuccess()
	return nil
}
