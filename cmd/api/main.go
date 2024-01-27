package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/oliverbenns/social-feed/internal/server/api"
	redis "github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	redisClient, err := createRedisClient(ctx)
	if err != nil {
		logger.Error("error connecting to redis", "error", err)
		os.Exit(1)
	}

	svc := api.Service{
		Port:            8080,
		RedisClient:     redisClient,
		Logger:          logger,
		InstagramAppID:  os.Getenv("INSTAGRAM_APP_ID"),
		InstagramSecret: os.Getenv("INSTAGRAM_SECRET"),
		AppURL:          os.Getenv("APP_URL"),
	}

	err = svc.Run(ctx)
	if err != nil {
		logger.Error("error running service", "error", err)
		os.Exit(1)
	}
}

func createRedisClient(ctx context.Context) (*redis.Client, error) {
	redisUrl := os.Getenv("REDIS_URL")

	opt, err := redis.ParseURL(redisUrl)
	if err != nil {
		return nil, fmt.Errorf("redis url parse failed: %w", err)
	}

	redisClient := redis.NewClient(opt)

	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return redisClient, nil
}
