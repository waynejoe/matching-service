package redisx

import "github.com/redis/go-redis/v9"

// NewClient 创建 Redis 客户端。
func NewClient(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}
