package db

import (
	"context"
	"github.com/sio2project/ft-to-s3/v1/utils"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client = nil
var redisContext = context.Background()

func StartClient(redisConfig *utils.RedisConfig) {
	rdb = redis.NewClient(&redis.Options{
		Addr:     redisConfig.Address,
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
	})

	utils.MainLogger.Info("Connected to Redis")
}
