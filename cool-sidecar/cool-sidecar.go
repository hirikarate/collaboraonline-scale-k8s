package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ctx        = context.Background()
	redisAddr  = os.Getenv("REDIS_ADDR") // e.g. "redis:6379"
	redisPass  = os.Getenv("REDIS_PASS") // optional
	podIP      = os.Getenv("POD_IP")     // injected by Downward API
	keyTTL     = 30 * time.Second        // heartbeat TTL
	heartbeat  = 10 * time.Second        // refresh interval
	registryNS = "collabora:pods"
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPass,
		DB:       0,
	})

	key := fmt.Sprintf("%s:%s", registryNS, podIP)

	// Register on startup
	if err := rdb.Set(ctx, key, "1", keyTTL).Err(); err != nil {
		panic(err)
	}
	fmt.Println("Registered pod:", key)

	// Start heartbeat to refresh TTL
	ticker := time.NewTicker(heartbeat)
	defer ticker.Stop()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		for range ticker.C {
			rdb.Set(ctx, key, "1", keyTTL)
		}
	}()

	// Wait for termination
	<-stop
	fmt.Println("Deregistering pod:", key)

	if err := rdb.Del(ctx, key).Err(); err != nil {
		fmt.Println("Failed to deregister:", err)
	} else {
		fmt.Println("Deregistered successfully")
	}
}
