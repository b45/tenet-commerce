package redis

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
)

// Client wraps the redis.Client with helper methods
type Client struct {
	RDB *redis.Client
}

// NewRedisClient initializes and verifies connection to the Redis server
func NewRedisClient(ctx context.Context) (*Client, error) {
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	password := os.Getenv("REDIS_PASSWORD")
	dbStr := os.Getenv("REDIS_DB")

	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "6379"
	}

	db := 0
	if dbStr != "" {
		if parsedDB, err := strconv.Atoi(dbStr); err == nil {
			db = parsedDB
		}
	}

	addr := fmt.Sprintf("%s:%s", host, port)

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     20,
		MinIdleConns: 5,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Ping to verify connection
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis at %s: %w", addr, err)
	}

	logger.Info("Successfully connected to Redis",
		"addr", addr,
		"db", db,
		"pool_size", 20,
	)

	return &Client{RDB: rdb}, nil
}

// Close closes the Redis connection pool
func (c *Client) Close() error {
	if c.RDB != nil {
		logger.Info("Closing Redis connection pool")
		return c.RDB.Close()
	}
	return nil
}
