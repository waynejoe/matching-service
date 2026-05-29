package redisx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const unlockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`

// Lock 是 Redis 分布式锁（SET NX + Lua 安全释放）。
type Lock struct {
	client *redis.Client
	prefix string
}

// NewLock 创建带 key 前缀的分布式锁。
func NewLock(client *redis.Client, keyPrefix string) *Lock {
	return &Lock{client: client, prefix: keyPrefix}
}

// WithLock 获取锁后执行 fn，结束后释放。
func (l *Lock) WithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	lockKey := l.prefix + key
	token, err := randomToken()
	if err != nil {
		return err
	}
	if err := l.waitLock(ctx, lockKey, token, ttl); err != nil {
		return err
	}
	defer l.unlock(context.Background(), lockKey, token)
	return fn()
}

func (l *Lock) waitLock(ctx context.Context, key, token string, ttl time.Duration) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		ok, err := l.client.SetNX(ctx, key, token, ttl).Result()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (l *Lock) unlock(ctx context.Context, key, token string) {
	_ = l.client.Eval(ctx, unlockScript, []string{key}, token).Err()
}

// Close 关闭底层 Redis 客户端。
func (l *Lock) Close() error {
	if l == nil || l.client == nil {
		return nil
	}
	return l.client.Close()
}

// Ping 检查 Redis 是否可用。
func (l *Lock) Ping(ctx context.Context) error {
	if l == nil || l.client == nil {
		return errors.New("redis client is nil")
	}
	return l.client.Ping(ctx).Err()
}

func randomToken() (string, error) {
	bs := make([]byte, 16)
	n, err := rand.Read(bs)
	if err != nil {
		return "", err
	}
	if n != len(bs) {
		return "", errors.New("lock token generation failed")
	}
	return hex.EncodeToString(bs), nil
}
