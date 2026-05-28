package server

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"

	"matching-service/internal/conf"
	"matching-service/internal/service"
)

// MessageHandler 是 RocketMQ 消息处理函数。
type MessageHandler func(ctx context.Context, body []byte) error

// RocketMQConsumer 管理撮合服务的 RocketMQ 顺序消费。
type RocketMQConsumer struct {
	cfg      conf.RocketMQ             // cfg 是 RocketMQ 消费配置
	consumer *service.MatchingConsumer // consumer 是撮合消息处理入口
	client   rocketmq.PushConsumer     // client 是 RocketMQ push consumer
}

// NewRocketMQConsumer 根据配置创建 RocketMQ 消费管理器。
func NewRocketMQConsumer(cfg *conf.Bootstrap, matchingConsumer *service.MatchingConsumer) *RocketMQConsumer {
	return &RocketMQConsumer{
		cfg:      cfg.Data.RocketMQ,
		consumer: matchingConsumer,
	}
}

// Run 启动入金和出金 RocketMQ 顺序消费。
func (c *RocketMQConsumer) Run(ctx context.Context) error {
	if len(c.cfg.NameServers) == 0 {
		return errors.New("RocketMQ NameServer 为空")
	}
	if c.cfg.DepositTopic == "" || c.cfg.WithdrawTopic == "" {
		return errors.New("RocketMQ topic 为空")
	}
	mq, err := rocketmq.NewPushConsumer(
		consumer.WithGroupName(c.cfg.ConsumerGroup),
		consumer.WithNsResolver(primitive.NewPassthroughResolver(c.cfg.NameServers)),
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithConsumeFromWhere(consumer.ConsumeFromFirstOffset),
		consumer.WithConsumerOrder(true),
		consumer.WithMaxReconsumeTimes(c.cfg.MaxReconsumeTimes),
		consumer.WithSuspendCurrentQueueTimeMillis(time.Duration(c.cfg.SuspendQueueMillis)*time.Millisecond),
	)
	if err != nil {
		return err
	}
	c.client = mq
	if err := c.subscribe(c.cfg.DepositTopic, c.consumer.HandleDepositMessage); err != nil {
		return err
	}
	if err := c.subscribe(c.cfg.WithdrawTopic, c.consumer.HandleWithdrawMessage); err != nil {
		return err
	}
	if err := c.client.Start(); err != nil {
		return err
	}
	log.Printf("RocketMQ 顺序消费启动成功，group=%s，deposit=%s，withdraw=%s", c.cfg.ConsumerGroup, c.cfg.DepositTopic, c.cfg.WithdrawTopic)
	<-ctx.Done()
	return c.Shutdown()
}

// Shutdown 关闭 RocketMQ 消费。
func (c *RocketMQConsumer) Shutdown() error {
	if c.client == nil {
		return nil
	}
	return c.client.Shutdown()
}

// subscribe 订阅 RocketMQ topic。
func (c *RocketMQConsumer) subscribe(topic string, handler MessageHandler) error {
	return c.client.Subscribe(topic, consumer.MessageSelector{}, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		for _, msg := range msgs {
			if err := handler(ctx, msg.Body); err != nil {
				log.Printf("处理 RocketMQ 消息失败，topic=%s，queue=%d，offset=%d，reconsume=%d，err=%v", msg.Topic, msg.Queue.QueueId, msg.QueueOffset, msg.ReconsumeTimes, err)
				return consumer.SuspendCurrentQueueAMoment, nil
			}
		}
		return consumer.ConsumeSuccess, nil
	})
}
