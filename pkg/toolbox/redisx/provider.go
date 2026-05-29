package redisx

import (
	"matching-service/internal/conf"

	"github.com/google/wire"
)

// ProviderSet 供 Wire 注入撮合分片锁。
var ProviderSet = wire.NewSet(NewShardLockProvider)

const defaultShardLockPrefix = "matching:shard:"

// NewShardLock 按 Bootstrap 创建分片锁（调用方负责 Close）。
func NewShardLock(cfg *conf.Bootstrap) *Lock {
	client := NewClient(cfg.Data.Redis.Addr, cfg.Data.Redis.Password, int(cfg.Data.Redis.Db))
	return NewLock(client, defaultShardLockPrefix)
}

// NewShardLockProvider 按 Bootstrap 配置创建分片锁并在 cleanup 时关闭连接。
func NewShardLockProvider(cfg *conf.Bootstrap) (*Lock, func(), error) {
	l := NewShardLock(cfg)
	return l, func() { _ = l.Close() }, nil
}
