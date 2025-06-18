package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client
var Ctx = context.Background()

func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
}

type SendMessageRequest struct {
	UserID int64  `json:"user_id"`
	Text   string `json:"text"`
}

type BotRequest struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type UserExistsRequest struct {
	UserID int64 `json:"user_id"`
}

type UserSingUpRequest struct {
	UserID int64  `json:"user_id"`
	Phone  string `json:"phone"`
	Name   string `json:"name"`
}

// Универсальная функция pub/sub-запроса с ожиданием ответа
func SendRequestAndWaitResponse(requestChannel, responseKey string, payload interface{}, timeout time.Duration) (string, error) {
	data, _ := json.Marshal(payload)

	err := RedisClient.Publish(Ctx, requestChannel, data).Err()
	if err != nil {
		return "", err
	}

	res, err := RedisClient.BLPop(Ctx, timeout, responseKey).Result()
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

// CheckUserExistsPubSub отправляет запрос на проверку пользователя и ждёт ответ через универсальную функцию
func CheckUserExistsPubSub(userID int64) (bool, error) {
	req := BotRequest{
		Type: "user_exists",
		Data: UserExistsRequest{UserID: userID},
	}
	key := fmt.Sprintf("user_exists_response:%d", userID)
	resp, err := SendRequestAndWaitResponse("bot_requests", key, req, 2*time.Second)
	if err != nil {
		return false, err
	}
	if resp == "true" {
		return true, nil
	}
	if resp == "false" {
		return false, nil
	}
	fmt.Println("Error checking user existence:", "unexpected response", resp)
	return false, fmt.Errorf("unexpected response: %s", resp)
}

func StartRequestListener(onMessageRecieved func(req SendMessageRequest)) {
	fmt.Println("Starting request listener")

	pubsub := RedisClient.Subscribe(Ctx, "api_requests")
	ch := pubsub.Channel()

	go func() {
		for msg := range ch {
			var req SendMessageRequest
			if err := json.Unmarshal([]byte(msg.Payload), &req); err != nil {
				fmt.Println("Error unmarshalling bot request:", err)
				continue
			}

			fmt.Println("msg.Payload", msg.Payload)

			onMessageRecieved(req)
		}
	}()
}
