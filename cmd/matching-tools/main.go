package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"matching-service/internal/conf"
	"matching-service/internal/server"
)

// replayMessage 是重放工具使用的通用消息结构。
type replayMessage struct {
	EventID string          `json:"eventId"` // EventID 是事件唯一 ID
	Topic   string          `json:"topic"`   // Topic 是 RocketMQ 主题
	Data    json.RawMessage `json:"data"`    // Data 是原始业务数据
}

// replayData 是重放工具提取分片键使用的数据结构。
type replayData struct {
	Channel  string `json:"channel"`  // Channel 是支付渠道
	Currency string `json:"currency"` // Currency 是币种
}

// main 是工具命令入口。
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run 解析命令并执行工具逻辑。
func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("缺少命令，可用命令：replay")
	}
	switch os.Args[1] {
	case "replay":
		return runReplay(os.Args[2:])
	default:
		return fmt.Errorf("未知命令：%s", os.Args[1])
	}
}

// runReplay 重投 RocketMQ 消息。
func runReplay(args []string) error {
	fs := flag.NewFlagSet("replay", flag.ExitOnError)
	confPath := fs.String("conf", "./configs/config.yaml", "配置文件路径")
	kind := fs.String("kind", "", "消息类型：deposit 或 withdraw")
	bodyFile := fs.String("body-file", "", "消息 body 文件路径")
	eventID := fs.String("event-id", "", "覆盖事件 ID，可选")
	delayLevel := fs.Int("delay-level", 0, "RocketMQ 延迟级别，可选")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *bodyFile == "" {
		return fmt.Errorf("body-file 不能为空")
	}
	cfg, err := loadBootstrap(*confPath)
	if err != nil {
		return err
	}
	body, msg, err := loadReplayBody(*bodyFile, *eventID)
	if err != nil {
		return err
	}
	channel, currency, err := parseReplayShard(msg.Data)
	if err != nil {
		return err
	}
	topic, err := replayTopic(cfg, *kind, msg.Topic)
	if err != nil {
		return err
	}
	producer, err := server.NewRocketMQProducer(cfg.Data.Rocketmq)
	if err != nil {
		return err
	}
	if err := producer.Start(); err != nil {
		return err
	}
	defer func() {
		_ = producer.Shutdown()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := producer.SendRaw(ctx, topic, msg.EventID, channel, currency, body, *delayLevel); err != nil {
		return err
	}
	fmt.Printf("重投成功，topic=%s，event_id=%s，sharding_key=%s:%s\n", topic, msg.EventID, channel, currency)
	return nil
}

// loadReplayBody 读取并按需改写消息体。
func loadReplayBody(path, eventID string) ([]byte, replayMessage, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, replayMessage{}, err
	}
	var msg replayMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, replayMessage{}, err
	}
	if eventID != "" {
		msg.EventID = eventID
		body, err = json.Marshal(msg)
		if err != nil {
			return nil, replayMessage{}, err
		}
	}
	if msg.EventID == "" {
		return nil, replayMessage{}, fmt.Errorf("eventId 不能为空")
	}
	return body, msg, nil
}

// parseReplayShard 从消息 data 提取分片键。
func parseReplayShard(data json.RawMessage) (string, string, error) {
	var out replayData
	if err := json.Unmarshal(data, &out); err != nil {
		return "", "", err
	}
	if out.Channel == "" || out.Currency == "" {
		return "", "", fmt.Errorf("消息 data.channel 和 data.currency 不能为空")
	}
	return out.Channel, out.Currency, nil
}

// replayTopic 返回重投目标 topic。
func replayTopic(cfg *conf.Bootstrap, kind, fallback string) (string, error) {
	switch kind {
	case "deposit":
		return cfg.Data.Rocketmq.DepositTopic, nil
	case "withdraw":
		return cfg.Data.Rocketmq.WithdrawTopic, nil
	case "":
		if fallback == "" {
			return "", fmt.Errorf("kind 和消息 topic 不能同时为空")
		}
		return fallback, nil
	default:
		return "", fmt.Errorf("kind 只能是 deposit 或 withdraw")
	}
}
