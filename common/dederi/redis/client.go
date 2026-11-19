package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	"github.com/redis/go-redis/v9"
	"strings"
)

type RedissonClient struct {
	Mode   string
	Client redis.UniversalClient
}

func NewRedisClient(setting *conf.Redis) redis.UniversalClient {
	if setting.PoolSize == 0 {
		setting.PoolSize = 10
	}

	if setting.MinIdle == 0 {
		setting.MinIdle = 5
	}

	// 构建 Redis 地址
	addrs := []string{}
	if setting.Port > 0 {
		// 如果配置了端口，使用 host:port 格式
		hosts := strings.Split(setting.Host, ",")
		for _, host := range hosts {
			addrs = append(addrs, fmt.Sprintf("%s:%d", host, setting.Port))
		}
	} else {
		// 如果没配置端口，使用原来的格式
		addrs = strings.Split(setting.Host, ",")
	}

	opt := &redis.UniversalOptions{
		Addrs:        addrs,
		Username:     setting.UserName,
		Password:     setting.Password,
		DB:           setting.DB,
		PoolSize:     setting.PoolSize,
		MinIdleConns: setting.MinIdle,
	}
	if setting.TLS {
		opt.TLSConfig = &tls.Config{}
	}

	universalClient := redis.NewUniversalClient(opt)

	_, pingErr := universalClient.Ping(context.Background()).Result()
	if pingErr != nil {
		panic(fmt.Sprintf("redis init error: %v", pingErr))
	}

	return universalClient
}
