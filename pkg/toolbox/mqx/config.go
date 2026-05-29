package mqx

import (
	"github.com/apache/rocketmq-clients/golang/v5/credentials"
)

// Config 是 RocketMQ 5.x 连接配置。
type Config struct {
	Endpoint     string
	NameSpace    string
	AccessKey    string
	AccessSecret string
}

func (c *Config) buildSessionCredentials() *credentials.SessionCredentials {
	if c == nil {
		return &credentials.SessionCredentials{}
	}
	return &credentials.SessionCredentials{
		AccessKey:    c.AccessKey,
		AccessSecret: c.AccessSecret,
	}
}

// ConsumerConfig 是消费者配置。
type ConsumerConfig struct {
	Group     string
	Topics    []string
	WorkerNum int32
	// Serial 为 true 时在接收循环内串行处理，保证同消费者内顺序语义。
	Serial bool
}

// NewConsumerConfig 创建单 topic 消费者配置。
func NewConsumerConfig(group, topic string, workerNum int32, serial bool) *ConsumerConfig {
	return NewConsumerConfigTopics(group, workerNum, serial, topic)
}

// NewConsumerConfigTopics 创建多 topic 消费者配置。
func NewConsumerConfigTopics(group string, workerNum int32, serial bool, topics ...string) *ConsumerConfig {
	return &ConsumerConfig{
		Group:     group,
		Topics:    topics,
		WorkerNum: workerNum,
		Serial:    serial,
	}
}
