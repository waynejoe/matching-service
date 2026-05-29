package mqx

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/apache/rocketmq-clients/golang/v5"
	v2 "github.com/apache/rocketmq-clients/golang/v5/protocol/v2"
	"github.com/panjf2000/ants/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"matching-service/pkg/toolbox/logx"
)

// MessageView 是 RocketMQ 消息视图。
type MessageView = golang.MessageView

const (
	awaitDuration     = time.Second * 5
	maxMessageNum     = int32(16)
	invisibleDuration = time.Second * 30
	maxWorkers        = 100
)

// ConsumerHandler 是消息处理函数。
type ConsumerHandler func(ctx context.Context, msg *MessageView) error

// ConsumerManager 管理多个消费者。
type ConsumerManager struct {
	consumers []*Consumer
}

// NewConsumerManager 创建消费者管理器。
func NewConsumerManager(consumers ...*Consumer) *ConsumerManager {
	return &ConsumerManager{consumers: consumers}
}

// Start 启动全部消费者。
func (m *ConsumerManager) Start(ctx context.Context) error {
	for _, consumer := range m.consumers {
		if err := consumer.Start(ctx); err != nil {
			logx.Fatalf(ctx, "start consumer failed: %v", err)
			return err
		}
	}
	return nil
}

// Stop 停止全部消费者。
func (m *ConsumerManager) Stop(ctx context.Context) error {
	for _, consumer := range m.consumers {
		if err := consumer.Stop(ctx); err != nil {
			logx.Errorf(ctx, "stop consumer failed: %v", err)
		}
	}
	return nil
}

// Consumer 是单个 topic 消费者。
type Consumer struct {
	consumer  golang.SimpleConsumer
	tagTopics []string
	handlers  map[string]ConsumerHandler
	mutx      sync.Mutex
	workerNum int
	serial    bool
}

// NewConsumer 创建消费者。
func NewConsumer(config *Config, consumerCfg *ConsumerConfig) (*Consumer, error) {
	if config == nil || consumerCfg == nil {
		return nil, fmt.Errorf("mqx: consumer config is nil")
	}
	if len(consumerCfg.Topics) == 0 {
		return nil, fmt.Errorf("mqx: topics is empty")
	}
	subscriptionExpressions := make(map[string]*golang.FilterExpression)
	for _, tagTopic := range consumerCfg.Topics {
		topic, tag := parseTagTopic(tagTopic)
		if tag != "" {
			subscriptionExpressions[topic] = golang.NewFilterExpression(tag)
		} else {
			subscriptionExpressions[topic] = golang.SUB_ALL
		}
	}
	simpleConsumer, err := golang.NewSimpleConsumer(
		&golang.Config{
			Endpoint:      config.Endpoint,
			NameSpace:     config.NameSpace,
			ConsumerGroup: consumerCfg.Group,
			Credentials:   config.buildSessionCredentials(),
		},
		golang.WithSimpleAwaitDuration(awaitDuration),
		golang.WithSimpleSubscriptionExpressions(subscriptionExpressions),
	)
	if err != nil {
		return nil, err
	}
	return &Consumer{
		consumer:  simpleConsumer,
		tagTopics: consumerCfg.Topics,
		handlers:  make(map[string]ConsumerHandler),
		workerNum: int(consumerCfg.WorkerNum),
		serial:    consumerCfg.Serial,
	}, nil
}

// Register 注册 topic/tag 处理器。
func (c *Consumer) Register(topic, tag string, handler ConsumerHandler) {
	c.mutx.Lock()
	defer c.mutx.Unlock()
	c.handlers[buildTopicKey(topic, tag)] = handler
}

// Start 启动消费循环。
func (c *Consumer) Start(ctx context.Context) error {
	if err := c.consumer.Start(); err != nil {
		return err
	}
	if c.serial {
		go c.receiveLoopSerial(ctx)
		return nil
	}
	if c.workerNum <= 0 {
		c.workerNum = maxWorkers
	}
	pool, err := ants.NewPool(c.workerNum)
	if err != nil {
		return fmt.Errorf("create ants pool: %w", err)
	}
	go c.receiveLoopPool(ctx, pool)
	return nil
}

func (c *Consumer) receiveLoopSerial(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			mvs, err := c.consumer.Receive(ctx, maxMessageNum, invisibleDuration)
			if err != nil {
				if isMessageNotFound(err) {
					continue
				}
				logx.Errorf(ctx, "receive message failed: %v", err)
				continue
			}
			for _, mv := range mvs {
				if mv == nil {
					continue
				}
				ctx2 := c.extractTrace(ctx, mv)
				handler, ok := c.getHandler(mv)
				if !ok {
					handler = ignoreConsumer
				}
				c.consumeMessage(ctx2, mv, handler)
			}
		}
	}
}

func (c *Consumer) receiveLoopPool(ctx context.Context, pool *ants.Pool) {
	defer pool.Release()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			mvs, err := c.consumer.Receive(ctx, maxMessageNum, invisibleDuration)
			if err != nil {
				if isMessageNotFound(err) {
					continue
				}
				logx.Errorf(ctx, "receive message failed: %v", err)
				continue
			}
			for _, mv := range mvs {
				if mv == nil {
					continue
				}
				ctx2 := c.extractTrace(ctx, mv)
				handler, ok := c.getHandler(mv)
				if !ok {
					handler = ignoreConsumer
				}
				mvCopy := mv
				hCopy := handler
				if err := pool.Submit(func() { c.consumeMessage(ctx2, mvCopy, hCopy) }); err != nil {
					logx.Errorf(ctx2, "submit consumer task failed: %v", err)
				}
			}
		}
	}
}

func (c *Consumer) consumeMessage(ctx context.Context, mv *MessageView, handler ConsumerHandler) {
	start := time.Now()
	defer func() {
		if r := recover(); r != nil {
			logx.Errorf(ctx, "panic in consumer topic=%s msg_id=%s: %v\n%s", mv.GetTopic(), mv.GetMessageId(), r, debug.Stack())
		}
	}()
	if err := handler(ctx, mv); err != nil {
		logx.Errorf(ctx, "handle message failed topic=%s msg_id=%s cost=%v err=%v", mv.GetTopic(), mv.GetMessageId(), time.Since(start), err)
		return
	}
	if err := c.consumer.Ack(ctx, mv); err != nil {
		logx.Errorf(ctx, "ack message failed topic=%s msg_id=%s err=%v", mv.GetTopic(), mv.GetMessageId(), err)
	}
}

// Stop 停止消费者。
func (c *Consumer) Stop(ctx context.Context) error {
	logx.Infof(ctx, "stop the consumer")
	return c.consumer.GracefulStop()
}

func (c *Consumer) getHandler(msg *MessageView) (ConsumerHandler, bool) {
	var tag string
	if msg.GetTag() != nil {
		tag = *msg.GetTag()
	}
	handler, ok := c.handlers[buildTopicKey(msg.GetTopic(), tag)]
	return handler, ok
}

func (c *Consumer) extractTrace(ctx context.Context, msg *MessageView) context.Context {
	carrier := propagation.MapCarrier(msg.GetProperties())
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

func isMessageNotFound(err error) bool {
	var mqErr *golang.ErrRpcStatus
	return errors.As(err, &mqErr) && v2.Code(mqErr.GetCode()) == v2.Code_MESSAGE_NOT_FOUND
}

func parseTagTopic(tagTopic string) (string, string) {
	parts := strings.Split(tagTopic, ":")
	if len(parts) == 1 {
		return strings.TrimSpace(parts[0]), ""
	}
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	panic(fmt.Sprintf("invalid tag topic: %s", tagTopic))
}

func buildTopicKey(topic, tag string) string {
	if tag == "" {
		return topic
	}
	return fmt.Sprintf("%s:%s", topic, tag)
}

func ignoreConsumer(ctx context.Context, msg *MessageView) error {
	logx.Infof(ctx, "ignore message topic=%s msg_id=%s", msg.GetTopic(), msg.GetMessageId())
	return nil
}
