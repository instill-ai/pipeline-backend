package cache

import (
	"github.com/go-redis/redis/v8"

	"github.com/instill-ai/pipeline-backend/config"
)

var Redis *redis.Client

func Init() {
	Redis = redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
}

func Close() {
	if Redis != nil {
		Redis.Close()
	}
}
