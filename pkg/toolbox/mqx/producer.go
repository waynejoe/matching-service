package mqx

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/apache/rocketmq-clients/golang/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"matching-service/pkg/toolbox/logx"
)

func init() {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	_ = os.Setenv(string(golang.ENABLE_CONSOLE_APPENDER), "true")
	golang.ResetLogger()
}

// Producer 是 RocketMQ 5.x 生产者。
type Producer struct {
	producer golang.Producer
}

// NewProducer 创建生产者。
func NewProducer(config *Config) (*Producer, error) {
	if config == nil {
		return nil, fmt.Errorf("mqx: producer config is nil")
	}
	producer, err := golang.NewProducer(
		&golang.Config{
			Endpoint:    config.Endpoint,
			NameSpace:   config.NameSpace,
			Credentials: config.buildSessionCredentials(),
		},
	)
	if err != nil {
		return nil, err
	}
	return &Producer{producer: producer}, nil
}

// Start 启动生产者。
func (p *Producer) Start(ctx context.Context) error {
	_ = ctx
	return p.producer.Start()
}

// Stop 停止生产者。
func (p *Producer) Stop(ctx context.Context) error {
	logx.Infof(ctx, "stop the producer")
	return p.producer.GracefulStop()
}

// Send 发送同步消息。
func (p *Producer) Send(ctx context.Context, data *Message) error {
	if data == nil || data.GetBody() == nil {
		return nil
	}
	msg, ctx2 := p.buildMessage(ctx, data)
	if _, err := p.producer.Send(ctx2, msg); err != nil {
		logx.Errorf(ctx2, "send message error: %v, topic: %s", err, data.Topic)
		return err
	}
	return nil
}

func (p *Producer) buildMessage(ctx context.Context, data *Message) (*golang.Message, context.Context) {
	msg := &golang.Message{Topic: data.Topic, Body: data.GetBody()}
	ctx = p.injectTrace(ctx, msg)
	if data.Tag != "" {
		msg.SetTag(data.Tag)
	}
	if data.MessageGroup != "" {
		msg.SetMessageGroup(data.MessageGroup)
	}
	if data.Duration > 0 {
		msg.SetDelayTimestamp(time.Now().Add(data.Duration))
	}
	if data.Key != "" {
		msg.SetKeys(data.Key)
	} else if traceID := trace.SpanContextFromContext(ctx).TraceID().String(); traceID != "" {
		msg.SetKeys(traceID)
	}
	return msg, ctx
}

func (p *Producer) injectTrace(ctx context.Context, msg *golang.Message) context.Context {
	ctx, span := otel.Tracer("mq").Start(ctx, "injectTrace")
	defer span.End()
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	for k, v := range carrier {
		msg.AddProperty(k, v)
	}
	return ctx
}
