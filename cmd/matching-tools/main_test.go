package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"matching-service/internal/conf"
)

// TestLoadReplayBodyOverrideEventID 验证重放工具可以覆盖事件 ID。
func TestLoadReplayBodyOverrideEventID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "message.json")
	body := `{"eventId":"old","topic":"match_deposit","data":{"channel":"bank","currency":"CNY"}}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("写入测试消息失败: %v", err)
	}
	out, msg, err := loadReplayBody(path, "new")
	if err != nil {
		t.Fatalf("读取重放消息失败: %v", err)
	}
	if msg.EventID != "new" {
		t.Fatalf("期望事件 ID 被覆盖为 new，实际 %s", msg.EventID)
	}
	if !strings.Contains(string(out), `"eventId":"new"`) {
		t.Fatalf("期望消息体包含新事件 ID，实际 %s", string(out))
	}
}

// TestParseReplayShard 验证重放工具可以提取 RocketMQ 分片键。
func TestParseReplayShard(t *testing.T) {
	channel, currency, err := parseReplayShard([]byte(`{"channel":"bank","currency":"CNY"}`))
	if err != nil {
		t.Fatalf("提取分片键失败: %v", err)
	}
	if channel != "bank" || currency != "CNY" {
		t.Fatalf("分片键不正确: %s:%s", channel, currency)
	}
}

// TestReplayTopic 验证重放 topic 选择规则。
func TestReplayTopic(t *testing.T) {
	cfg := &conf.Bootstrap{}
	cfg.Data.RocketMQ.DepositTopic = "match_deposit"
	cfg.Data.RocketMQ.WithdrawTopic = "match_withdraw"
	topic, err := replayTopic(cfg, "deposit", "")
	if err != nil {
		t.Fatalf("选择入金 topic 失败: %v", err)
	}
	if topic != "match_deposit" {
		t.Fatalf("入金 topic 不正确: %s", topic)
	}
	topic, err = replayTopic(cfg, "", "custom_topic")
	if err != nil {
		t.Fatalf("选择兜底 topic 失败: %v", err)
	}
	if topic != "custom_topic" {
		t.Fatalf("兜底 topic 不正确: %s", topic)
	}
}
