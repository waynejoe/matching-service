package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"google.golang.org/protobuf/types/known/timestamppb"

	"matching-service/internal/conf"
	"matching-service/internal/model"
	v1 "matching-service/api/matching/v1"
)

// RocketMQProducer 负责按分片 key 发送 RocketMQ 消息。
type RocketMQProducer struct {
	cfg      conf.RocketMQ     // cfg 是 RocketMQ 配置
	producer rocketmq.Producer // producer 是 RocketMQ producer
}

// NewRocketMQProducer 创建 RocketMQ 生产者。
func NewRocketMQProducer(cfg conf.RocketMQ) (*RocketMQProducer, error) {
	p, err := rocketmq.NewProducer(
		producer.WithNsResolver(primitive.NewPassthroughResolver(cfg.NameServers)),
		producer.WithQueueSelector(producer.NewHashQueueSelector()),
		producer.WithRetry(2),
	)
	if err != nil {
		return nil, err
	}
	return &RocketMQProducer{cfg: cfg, producer: p}, nil
}

// Start 启动 RocketMQ 生产者。
func (p *RocketMQProducer) Start() error {
	return p.producer.Start()
}

// Shutdown 关闭 RocketMQ 生产者。
func (p *RocketMQProducer) Shutdown() error {
	return p.producer.Shutdown()
}

// SendDeposit 发送入金主队列消息。
func (p *RocketMQProducer) SendDeposit(ctx context.Context, eventID string, deposit model.DepositOrder, delayLevel int) error {
	msg, err := p.buildMessage(p.cfg.DepositTopic, eventID, deposit.Channel, deposit.Currency, &v1.DepositEventMessage{
		EventId: eventID,
		Topic:   p.cfg.DepositTopic,
		Data:    depositToPB(deposit),
	}, delayLevel)
	if err != nil {
		return err
	}
	_, err = p.producer.SendSync(ctx, msg)
	return err
}

// SendWithdraw 发送出金主队列消息。
func (p *RocketMQProducer) SendWithdraw(ctx context.Context, eventID string, basket model.WithdrawBasket, delayLevel int) error {
	msg, err := p.buildMessage(p.cfg.WithdrawTopic, eventID, basket.Channel, basket.Currency, &v1.WithdrawEventMessage{
		EventId: eventID,
		Topic:   p.cfg.WithdrawTopic,
		Data:    basketToPB(basket),
	}, delayLevel)
	if err != nil {
		return err
	}
	_, err = p.producer.SendSync(ctx, msg)
	return err
}

// SendRaw 发送已经编码好的 RocketMQ 消息体。
func (p *RocketMQProducer) SendRaw(ctx context.Context, topic, eventID, channel, currency string, body []byte, delayLevel int) error {
	msg := primitive.NewMessage(topic, body)
	msg.WithKeys([]string{eventID})
	msg.WithShardingKey(fmt.Sprintf("%s:%s", channel, currency))
	if delayLevel > 0 {
		msg.WithDelayTimeLevel(delayLevel)
	}
	_, err := p.producer.SendSync(ctx, msg)
	return err
}

// buildMessage 构造带分片 key 和可选延迟级别的 RocketMQ 消息。
func (p *RocketMQProducer) buildMessage(topic, eventID, channel, currency string, payload any, delayLevel int) (*primitive.Message, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	msg := primitive.NewMessage(topic, body)
	msg.WithKeys([]string{eventID})
	msg.WithShardingKey(fmt.Sprintf("%s:%s", channel, currency))
	if delayLevel > 0 {
		msg.WithDelayTimeLevel(delayLevel)
	}
	return msg, nil
}

// depositToPB 把 model 入金单转换成 proto 入金单。
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

// basketToPB 把 model 出金篮子转换成 proto 出金篮子。
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

// timeToTimestamp 把 Go 时间转换成 protobuf 时间戳。
func timeToTimestamp(in time.Time) *timestamppb.Timestamp {
	if in.IsZero() {
		return nil
	}
	return timestamppb.New(in)
}
