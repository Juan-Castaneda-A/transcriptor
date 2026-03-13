package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Juan-Castaneda-A/transcriptor/internal/models"
	"github.com/redis/go-redis/v9"
)

const (
	// QueueName is the Redis list used as the message queue.
	QueueName = "transcription:jobs"
	// StatusChannel is the Redis pub/sub channel for status updates.
	StatusChannel = "transcription:status"
)

// Client wraps Redis operations for the message queue.
type Client struct {
	rdb *redis.Client
}

// NewClient creates a new queue client from a Redis URL.
func NewClient(redisURL string) (*Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	rdb := redis.NewClient(opts)

	// Test connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("✅ Connected to Redis")
	return &Client{rdb: rdb}, nil
}

// Enqueue pushes a transcription job onto the queue.
func (c *Client) Enqueue(ctx context.Context, msg models.QueueMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal queue message: %w", err)
	}

	if err := c.rdb.LPush(ctx, QueueName, data).Err(); err != nil {
		return fmt.Errorf("failed to enqueue message: %w", err)
	}

	log.Printf("📨 Enqueued job for file: %s", msg.FileID)
	return nil
}

// PublishStatus sends a status update via Redis pub/sub.
// The frontend listens to these via WebSocket.
func (c *Client) PublishStatus(ctx context.Context, fileID, status, message string) error {
	update := map[string]string{
		"file_id": fileID,
		"status":  status,
		"message": message,
	}
	data, _ := json.Marshal(update)
	return c.rdb.Publish(ctx, StatusChannel, data).Err()
}

// Subscribe listens for status updates on the Redis pub/sub channel.
func (c *Client) Subscribe(ctx context.Context) *redis.PubSub {
	return c.rdb.Subscribe(ctx, StatusChannel)
}

// RawClient returns the underlying redis client.
func (c *Client) RawClient() *redis.Client {
	return c.rdb
}

// Close cleanly shuts down the Redis connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}
