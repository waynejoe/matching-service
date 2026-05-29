package server

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"

	"matching-service/internal/conf"
	"matching-service/internal/service"
	"matching-service/pkg/toolbox/mqx"
)

func mqConfig(c *conf.RocketMQ) *mqx.Config {
	return &mqx.Config{
		Endpoint:     c.GetEndpoint(),
		NameSpace:    c.GetNamespace(),
		AccessKey:    c.GetAccessKey(),
		AccessSecret: c.GetAccessSecret(),
	}
}

// NewConsumerManager 创建入金/出金 RocketMQ 消费者（串行模式保序）。
func NewConsumerManager(c *conf.Bootstrap, consumer *service.MatchingConsumer, logger log.Logger) (*mqx.ConsumerManager, error) {
	mq := c.GetData().GetRocketmq()
	if mq == nil {
		return nil, fmt.Errorf("rocketmq config is nil")
	}
	if mq.GetEndpoint() == "" {
		return nil, fmt.Errorf("rocketmq endpoint is empty")
	}
	if mq.GetDepositTopic() == "" || mq.GetWithdrawTopic() == "" {
		return nil, fmt.Errorf("rocketmq topic is empty")
	}
	cons := mq.GetConsumer()
	if cons == nil || cons.GetGroup() == "" {
		return nil, fmt.Errorf("rocketmq consumer group is empty")
	}
	workerNum := cons.GetWorkerNum()
	if workerNum <= 0 {
		workerNum = 1
	}
	serial := cons.GetSerial()
	mqConsumer, err := mqx.NewConsumer(mqConfig(mq), mqx.NewConsumerConfigTopics(
		cons.GetGroup(),
		workerNum,
		serial,
		mq.GetDepositTopic(),
		mq.GetWithdrawTopic(),
	))
	if err != nil {
		return nil, err
	}
	depositHandler := func(ctx context.Context, msg *mqx.MessageView) error {
		return consumer.HandleDepositMessage(ctx, msg.GetBody())
	}
	withdrawHandler := func(ctx context.Context, msg *mqx.MessageView) error {
		return consumer.HandleWithdrawMessage(ctx, msg.GetBody())
	}
	mqConsumer.Register(mq.GetDepositTopic(), "", depositHandler)
	mqConsumer.Register(mq.GetWithdrawTopic(), "", withdrawHandler)
	_ = logger
	return mqx.NewConsumerManager(mqConsumer), nil
}

// NewMQProducer 创建 RocketMQ 5.x 生产者并注册到 Kratos 生命周期。
func NewMQProducer(c *conf.Bootstrap) (*mqx.Producer, error) {
	mq := c.GetData().GetRocketmq()
	if mq == nil || mq.GetEndpoint() == "" {
		return nil, fmt.Errorf("rocketmq config is invalid")
	}
	return mqx.NewProducer(mqConfig(mq))
}
