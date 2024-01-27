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
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	redisClient, err := createRedisClient(ctx)
	if err != nil {
		return fmt.Errorf("error connecting to redis: %w", err)
	}

	instagramAppID := os.Getenv("INSTAGRAM_APP_ID")
	if instagramAppID == "" {
		return fmt.Errorf("INSTAGRAM_APP_ID not set")
	}

	instagramSecret := os.Getenv("INSTAGRAM_SECRET")
	if instagramSecret == "" {
		return fmt.Errorf("INSTAGRAM_SECRET not set")
	}

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		return fmt.Errorf("APP_URL not set")
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return fmt.Errorf("API_KEY not set")
	}

	svc := api.Service{
		Port:            8080,
		RedisClient:     redisClient,
		Logger:          logger,
		InstagramAppID:  instagramAppID,
		InstagramSecret: instagramSecret,
		AppURL:          appURL,
		ApiKey:          apiKey,
	}

	err = svc.Run(ctx)
	if err != nil {
		return fmt.Errorf("error running service: %w", err)
	}

	return nil
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
