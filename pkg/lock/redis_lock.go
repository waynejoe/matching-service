package lock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"matching-service/internal/conf"
)

const unlockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`

// RedisLock 是 Redis 分布式锁。
type RedisLock struct {
	client *redis.Client // client 是 Redis 客户端
	prefix string        // prefix 是锁 key 前缀
}

// NewRedisClient 创建 Redis 客户端。
func NewRedisClient(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

// NewRedisLock 根据配置创建 Redis 分片锁。
func NewRedisLock(cfg *conf.Bootstrap) *RedisLock {
	client := NewRedisClient(cfg.Data.Redis.Addr, cfg.Data.Redis.Password, cfg.Data.Redis.DB)
	return &RedisLock{
		client: client,
		prefix: "matching:shard:",
	}
}

// WithLock 获取锁后执行函数，执行结束后释放锁。
func (l *RedisLock) WithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
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

// waitLock 等待并获取锁。
func (l *RedisLock) waitLock(ctx context.Context, key, token string, ttl time.Duration) error {
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

// unlock 只释放当前持有者的锁。
func (l *RedisLock) unlock(ctx context.Context, key, token string) {
	_ = l.client.Eval(ctx, unlockScript, []string{key}, token).Err()
}

// Close 关闭 Redis 客户端。
func (l *RedisLock) Close() error {
	if l == nil || l.client == nil {
		return nil
	}
	return l.client.Close()
}

// Ping 检查 Redis 连接是否可用。
func (l *RedisLock) Ping(ctx context.Context) error {
	if l == nil || l.client == nil {
		return errors.New("Redis 客户端为空")
	}
	return l.client.Ping(ctx).Err()
}

// randomToken 生成锁持有者令牌。
func randomToken() (string, error) {
	bs := make([]byte, 16)
	n, err := rand.Read(bs)
	if err != nil {
		return "", err
	}
	if n != len(bs) {
		return "", errors.New("生成锁令牌失败")
	}
	return hex.EncodeToString(bs), nil
}
