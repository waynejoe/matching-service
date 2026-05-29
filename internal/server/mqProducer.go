package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "matching-service/pkg/api/matching/v1"
	"matching-service/internal/conf"
	"matching-service/pkg/model"
	"matching-service/pkg/toolbox/mqx"
)

// RocketMQProducer 负责按 MessageGroup 发送 RocketMQ 5.x 消息。
type RocketMQProducer struct {
	cfg      *conf.RocketMQ
	producer *mqx.Producer
	own      bool
}

// NewRocketMQProducer 创建 RocketMQ 生产者（独立生命周期，供工具/健康检查使用）。
func NewRocketMQProducer(cfg *conf.RocketMQ) (*RocketMQProducer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("RocketMQ 配置为空")
	}
	if cfg.GetEndpoint() == "" {
		return nil, fmt.Errorf("RocketMQ endpoint 为空")
	}
	p, err := mqx.NewProducer(mqConfig(cfg))
	if err != nil {
		return nil, err
	}
	return &RocketMQProducer{cfg: cfg, producer: p, own: true}, nil
}

// WrapRocketMQProducer 包装已在 Kratos 中启动的共享生产者。
func WrapRocketMQProducer(cfg *conf.RocketMQ, p *mqx.Producer) *RocketMQProducer {
	return &RocketMQProducer{cfg: cfg, producer: p, own: false}
}

// Start 启动 RocketMQ 生产者。
func (p *RocketMQProducer) Start() error {
	return p.producer.Start(context.Background())
}

// Shutdown 关闭 RocketMQ 生产者。
func (p *RocketMQProducer) Shutdown() error {
	if !p.own {
		return nil
	}
	return p.producer.Stop(context.Background())
}

// SendDeposit 发送入金主队列消息。
func (p *RocketMQProducer) SendDeposit(ctx context.Context, eventID string, deposit model.DepositOrder, delayLevel int) error {
	payload := &v1.DepositEventMessage{
		EventId: eventID,
		Topic:   p.cfg.GetDepositTopic(),
		Data:    depositToPB(deposit),
	}
	return p.sendPayload(ctx, p.cfg.GetDepositTopic(), eventID, deposit.Channel, deposit.Currency, payload, delayLevel)
}

// SendWithdraw 发送出金主队列消息。
func (p *RocketMQProducer) SendWithdraw(ctx context.Context, eventID string, basket model.WithdrawBasket, delayLevel int) error {
	payload := &v1.WithdrawEventMessage{
		EventId: eventID,
		Topic:   p.cfg.GetWithdrawTopic(),
		Data:    basketToPB(basket),
	}
	return p.sendPayload(ctx, p.cfg.GetWithdrawTopic(), eventID, basket.Channel, basket.Currency, payload, delayLevel)
}

// SendRaw 发送已经编码好的 RocketMQ 消息体。
func (p *RocketMQProducer) SendRaw(ctx context.Context, topic, eventID, channel, currency string, body []byte, delayLevel int) error {
	msg := mqx.NewMessage(topic, body).
		WithKey(eventID).
		WithMessageGroup(shardKey(channel, currency))
	if d := delayLevelToDuration(delayLevel); d > 0 {
		msg.WithDelayDuration(d)
	}
	return p.producer.Send(ctx, msg)
}

func (p *RocketMQProducer) sendPayload(ctx context.Context, topic, eventID, channel, currency string, payload any, delayLevel int) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.SendRaw(ctx, topic, eventID, channel, currency, body, delayLevel)
}

func shardKey(channel, currency string) string {
	return fmt.Sprintf("%s:%s", channel, currency)
}

// delayLevelToDuration 将 RocketMQ v2 延迟级别映射为 v5 延迟时长。
func delayLevelToDuration(level int) time.Duration {
	table := []time.Duration{
		0,
		time.Second,
		5 * time.Second,
		10 * time.Second,
		30 * time.Second,
		time.Minute,
		2 * time.Minute,
		3 * time.Minute,
		4 * time.Minute,
		5 * time.Minute,
		6 * time.Minute,
		7 * time.Minute,
		8 * time.Minute,
		9 * time.Minute,
		10 * time.Minute,
		20 * time.Minute,
		30 * time.Minute,
		1 * time.Hour,
		2 * time.Hour,
	}
	if level <= 0 || level >= len(table) {
		return 0
	}
	return table[level]
}

func depositToPB(in model.DepositOrder) *v1.DepositOrder {
	return &v1.DepositOrder{
		DepositNo:       in.DepositNo,
		MerchantId:      in.MerchantID,
		Channel:         in.Channel,
		Currency:        in.Currency,
		Amount:          in.Amount,
		Status:          in.Status,
		ExpireAt:        timeToTimestamp(in.ExpireAt),
		MatchedBasketNo: in.MatchedBasketNo,
		MatchNo:         in.MatchNo,
		CreatedAt:       timeToTimestamp(in.CreatedAt),
		UpdatedAt:       timeToTimestamp(in.UpdatedAt),
	}
}

func basketToPB(in model.WithdrawBasket) *v1.WithdrawBasket {
	return &v1.WithdrawBasket{
		BasketNo:      in.BasketNo,
		WithdrawNo:    in.WithdrawNo,
		MerchantId:    in.MerchantID,
		Channel:       in.Channel,
		Currency:      in.Currency,
		TargetAmount:  in.TargetAmount,
		CurrentAmount: in.CurrentAmount,
		Status:        in.Status,
		ExpireAt:      timeToTimestamp(in.ExpireAt),
		Version:       in.Version,
		CreatedAt:     timeToTimestamp(in.CreatedAt),
		UpdatedAt:     timeToTimestamp(in.UpdatedAt),
	}
}

func timeToTimestamp(in time.Time) *timestamppb.Timestamp {
	if in.IsZero() {
		return nil
	}
	return timestamppb.New(in)
}
