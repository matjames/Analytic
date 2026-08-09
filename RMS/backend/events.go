package main

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

func initRedis() error {
	if os.Getenv("REDIS_HOST") == "" {
		return nil
	}
	redisClient = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Password: "",
		DB:       0,
	})
	ctx := context.Background()
	return redisClient.Ping(ctx).Err()
}

func publishEvent(eventType, objectType, objectID string, payload map[string]interface{}) {
	if redisClient == nil {
		return
	}
	ctx := context.Background()
	event := map[string]interface{}{
		"event_type":  eventType,
		"source":      "rms",
		"object_type": objectType,
		"object_id":   objectID,
		"tenant_id":   getEnv("RMS_TENANT_ID", "tenant-alpha"),
		"payload":     payload,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.Marshal(event)
	_ = redisClient.Publish(ctx, getEnv("STATGATE_EVENT_CHANNEL", "statgate:events"), string(data)).Err()
}
