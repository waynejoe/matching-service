package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"

	"matching-service/internal/conf"
	"matching-service/internal/model"
	"matching-service/internal/service"
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
	msg, err := p.buildMessage(p.cfg.DepositTopic, eventID, deposit.Channel, deposit.Currency, service.DepositEventMessage{
		EventID: eventID,
		Topic:   p.cfg.DepositTopic,
		Data:    deposit,
	}, delayLevel)
	if err != nil {
		return err
	}
	_, err = p.producer.SendSync(ctx, msg)
	return err
}

// SendWithdraw 发送出金主队列消息。
func (p *RocketMQProducer) SendWithdraw(ctx context.Context, eventID string, basket model.WithdrawBasket, delayLevel int) error {
	msg, err := p.buildMessage(p.cfg.WithdrawTopic, eventID, basket.Channel, basket.Currency, service.WithdrawEventMessage{
		EventID: eventID,
		Topic:   p.cfg.WithdrawTopic,
		Data:    basket,
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
