package db

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Address  string `json:"address"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

var rdb *redis.Client = nil
var redisContext = context.Background()

func StartClient(redisConfig *RedisConfig) {
	rdb = redis.NewClient(&redis.Options{
		Addr:     redisConfig.Address,
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
	})

	log.Println("Connected to Redis")
}
