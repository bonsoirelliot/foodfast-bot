package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	client *redis.Client
	ctx    context.Context
}

func New() *Client {
	client := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	return &Client{
		client: client,
		ctx:    context.Background(),
	}
}

func (c *Client) SendRequestAndWaitResponse(requestChannel, responseKey string, payload interface{}, timeout time.Duration) (string, error) {
	data, _ := json.Marshal(payload)

	err := c.client.Publish(c.ctx, requestChannel, data).Err()
	if err != nil {
		return "", err
	}

	res, err := c.client.BLPop(c.ctx, timeout, responseKey).Result()
	if err != nil {
		fmt.Println("Error in SendRequestAndWaitResponse:", err)
		if err.Error() == "redis: nil" {
			return "", fmt.Errorf("no response from service")
		}
		return "", err
	}
	if len(res) < 2 {
		fmt.Println("Error in SendRequestAndWaitResponse:", "invalid response from service")
		return "", fmt.Errorf("invalid response from service")
	}
	return res[1], nil
}

func (c *Client) Subscribe(channel string) *redis.PubSub {
	return c.client.Subscribe(c.ctx, channel)
}
