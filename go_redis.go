package redisutil

import (
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisSetting redis設定
type RedisSetting struct {
	// 接続先アドレス:ポート
	Server             string
	PoolSize           int
	DialTimeout        time.Duration
	PoolTimeout        time.Duration
	MinIdleConns       int
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	MaxConnAge         time.Duration
	IdleTimeout        time.Duration
	IdleCheckFrequency time.Duration
	MaxRetries         int
	MinRetryBackoff    time.Duration
	MaxRetryBackoff    time.Duration
}

func NewRedisClient(nodes []string) redis.UniversalClient {
	return redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:           nodes,
		PoolSize:        200,
		DialTimeout:     time.Second * 3,
		ReadTimeout:     time.Second * 5,
		WriteTimeout:    time.Second * 5,
		PoolTimeout:     time.Second * 5,
		MaxConnAge:      time.Second * 1800,
		MaxRetries:      3,
		MinRetryBackoff: time.Millisecond * 50,
		MaxRetryBackoff: time.Millisecond * 200,
	})
}
