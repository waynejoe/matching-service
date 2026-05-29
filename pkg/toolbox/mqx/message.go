package mqx

import (
	"encoding/json"
	"time"
)

// Message 是发送消息体。
type Message struct {
	Topic        string
	Tag          string
	Key          string
	MessageGroup string
	Duration     time.Duration
	Body         any
}

// NewMessage 构建消息。
func NewMessage(topic string, body any) *Message {
	return &Message{Topic: topic, Body: body}
}

func (m *Message) WithTag(tag string) *Message {
	m.Tag = tag
	return m
}

func (m *Message) WithKey(key string) *Message {
	m.Key = key
	return m
}

func (m *Message) WithMessageGroup(group string) *Message {
	m.MessageGroup = group
	return m
}

func (m *Message) WithDelayDuration(d time.Duration) *Message {
	m.Duration = d
	return m
}

func (m *Message) GetBody() []byte {
	if m == nil || m.Body == nil {
		return nil
	}
	switch body := m.Body.(type) {
	case []byte:
		return body
	case string:
		return []byte(body)
	default:
		data, err := json.Marshal(m.Body)
		if err != nil {
			return nil
		}
		return data
	}
}
