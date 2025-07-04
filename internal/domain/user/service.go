package user

import (
	"fmt"
	"time"

	"foodfast-bot/internal/domain/models"
)

type PubSubClient interface {
	SendRequestAndWaitResponse(requestChannel, responseKey string, payload interface{}, timeout time.Duration) (string, error)
}

type Service struct {
	pubSub PubSubClient
}

func New(pubSub PubSubClient) *Service {
	fmt.Println("New User Service")
	return &Service{pubSub: pubSub}
}

func (s *Service) CheckUserExists(userID int64) (bool, error) {
	req := models.BotRequest{
		Type: "user_exists",
		Data: models.UserExistsRequest{UserID: userID},
	}
	key := "user_exists_response"
	resp, err := s.pubSub.SendRequestAndWaitResponse("bot_requests", key, req, 2*time.Second)
	if err != nil {
		return false, err
	}
	return resp == "true", nil
}

func (s *Service) RegisterUser(userID int64, phone string, name string) error {
	req := models.BotRequest{
		Type: "sign_up",
		Data: models.UserSingUpRequest{UserID: userID, Phone: phone, Name: name},
	}
	_, err := s.pubSub.SendRequestAndWaitResponse("bot_requests", "sign_up_response", req, 2*time.Second)
	return err
}
