package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"foodfast-bot/internal/domain/models"
	"foodfast-bot/internal/domain/user"
	"foodfast-bot/internal/pkg/redis"
	"log"
	"net/http"
	"os"
)

type Bot struct {
	telegramToken string
	userService   *user.Service
	redisClient   *redis.Client
}

func New(userService *user.Service, redisClient *redis.Client) *Bot {
	return &Bot{
		telegramToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		userService:   userService,
		redisClient:   redisClient,
	}
}

func (b *Bot) Start() {
	fmt.Println("token: ", b.telegramToken)
	go b.startRequestListener()

	offset := 0
	for {
		updates := b.getUpdates(offset)
		for _, update := range updates {
			offset = update.UpdateID + 1
			b.handleUpdate(update)
		}
	}
}

func (b *Bot) getUpdates(offset int) []models.BotUpdate {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30", b.telegramToken, offset)
	resp, err := http.Get(url)
	if err != nil {
		log.Println("getUpdates error:", err)
		return nil
	}
	defer resp.Body.Close()
	var result struct {
		Result []models.BotUpdate `json:"result"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Result
}

func (b *Bot) handleUpdate(update models.BotUpdate) {
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID

	if update.Message.Text == "/start" {
		exists, err := b.userService.CheckUserExists(userID)
		if err != nil {
			b.SendMessage(chatID, "Ошибка при проверке пользователя. Попробуйте позже.")
			return
		}
		if exists {
			b.SendMessage(chatID, "Я пока что умею только регистрировать, новый функционал ещё в разработке...")
		} else {
			b.sendRequestPhone(chatID)
		}
		return
	}

	if update.Message.Contact.PhoneNumber != "" {
		err := b.userService.RegisterUser(userID, update.Message.Contact.PhoneNumber, update.Message.From.FirstName)
		if err != nil {
			b.SendMessage(chatID, "Ошибка при регистрации пользователя. Попробуйте позже.")
			return
		}
		b.SendMessage(chatID, "Спасибо! Вы зарегистрированы.")
		return
	}
}

func (b *Bot) SendMessage(chatID int64, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.telegramToken)
	body, _ := json.Marshal(map[string]interface{}{
		"chat_id":      chatID,
		"text":         text,
		"reply_markup": map[string]bool{"remove_keyboard": true},
	})
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("SendMessage error:", err)
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (b *Bot) sendRequestPhone(chatID int64) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.telegramToken)
	keyboard := map[string]interface{}{
		"keyboard": [][]map[string]interface{}{
			{
				{
					"text":            "📲 Отправить номер",
					"request_contact": true,
				},
			},
		},
		"resize_keyboard":   true,
		"one_time_keyboard": true,
	}
	body, _ := json.Marshal(map[string]interface{}{
		"chat_id":      chatID,
		"text":         "Привет! Для регистрации мне нужен только ваш номер телефона, остальное я сделаю сам 🤘\n\nНажмите на кнопочку «📲 Отправить номер», чтобы передать его.",
		"reply_markup": keyboard,
	})
	http.Post(url, "application/json", bytes.NewBuffer(body))
}

func (b *Bot) startRequestListener() {
	fmt.Println("Starting request listener")

	pubsub := b.redisClient.Subscribe("api_requests")
	ch := pubsub.Channel()

	for msg := range ch {
		var req models.SendMessageRequest
		if err := json.Unmarshal([]byte(msg.Payload), &req); err != nil {
			fmt.Println("Error unmarshalling bot request:", err)
			continue
		}
		fmt.Println("msg.Payload", msg.Payload)
		b.SendMessage(req.UserID, req.Text)
	}
}
