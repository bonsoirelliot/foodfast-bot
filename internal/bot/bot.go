package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"foodfast-bot/internal/domain/models"
	"foodfast-bot/internal/domain/user"
	"foodfast-bot/internal/pkg/rabbit"
	"log"
	"net/http"
	"os"
	"time"
)

type Bot struct {
	userService  *user.Service
	rabbitClient *rabbit.Client
}

func New(userService *user.Service, rabbitClient *rabbit.Client) *Bot {
	fmt.Println("New Bot Instance")
	return &Bot{
		userService:  userService,
		rabbitClient: rabbitClient,
	}
}

func (b *Bot) Start() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	go b.startRabbitListener(token)
	offset := 0
	for {
		updates := b.getUpdates(token, offset)
		for _, update := range updates {
			offset = update.UpdateID + 1
			log.Printf("[Polling] Received update: user_id=%d, text=%q", update.Message.From.ID, update.Message.Text)
			b.handleUpdate(token, update)
		}
	}
}

func (b *Bot) getUpdates(token string, offset int) []models.BotUpdate {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30", token, offset)
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

func (b *Bot) handleUpdate(token string, update models.BotUpdate) {
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID

	if update.Message.Text == "/start" {
		log.Printf("[Bot] /start from user_id=%d", userID)
		exists, err := b.checkUserExistsRabbit(userID)
		if err != nil {
			log.Printf("[Bot] Error checking user exists for user_id=%d: %v", userID, err)
			b.SendMessage(token, chatID, "Ошибка при проверке пользователя. Попробуйте позже.")
			return
		}
		if exists {
			log.Printf("[Bot] User %d exists, sending info message", userID)
			b.SendMessage(token, chatID, "Я пока что умею только регистрировать, новый функционал ещё в разработке...")
		} else {
			log.Printf("[Bot] User %d not found, requesting phone", userID)
			b.sendRequestPhone(token, chatID)
		}
		return
	}

	if update.Message.Contact.PhoneNumber != "" {
		log.Printf("[Bot] Got contact from user_id=%d, phone=%s", userID, update.Message.Contact.PhoneNumber)
		res, err := b.registerUserRabbit(userID, update.Message.Contact.PhoneNumber, update.Message.From.FirstName, update.Message.From.Username)
		if err != nil {
			log.Printf("[Bot] Error registering user_id=%d: %v", userID, err)
			b.SendMessage(token, chatID, "Ошибка при регистрации пользователя. Попробуйте позже.")
			return
		}
		if res {
			log.Printf("[Bot] User %d registered successfully", userID)
			b.SendMessage(token, chatID, "Спасибо! Вы зарегистрированы.")
			return
		} else {
			b.SendMessage(token, chatID, "Ошибка при регистрации пользователя. Попробуйте позже.")

		}

	}
}

func (b *Bot) checkUserExistsRabbit(userID int64) (bool, error) {
	log.Printf("[RabbitMQ] Sending user_exists for user_id=%d", userID)
	req := models.BotRequest{
		Type: "user_exists",
		Data: models.UserExistsRequest{UserID: userID},
	}
	resp, err := b.rabbitClient.SendRequestAndWaitResponse("bot_requests", req.Type, req, 2*time.Second)
	if err != nil {
		log.Printf("[RabbitMQ] Error waiting user_exists response for user_id=%d: %v", userID, err)
		return false, err
	}
	log.Printf("[RabbitMQ] user_exists response for user_id=%d: %q", userID, resp)
	return resp == "true", nil
}

func (b *Bot) registerUserRabbit(userID int64, phone, name, username string) (bool, error) {
	log.Printf("[RabbitMQ] Sending sign_up for user_id=%d, phone=%s, name=%s, username=%s", userID, phone, name, username)
	req := models.BotRequest{
		Type: "sign_up",
		Data: models.UserSingUpRequest{UserID: userID, Phone: phone, Name: name, Username: username},
	}
	resp, err := b.rabbitClient.SendRequestAndWaitResponse("bot_requests", req.Type, req, 2*time.Second)
	if err != nil {
		log.Printf("[RabbitMQ] Error waiting sign_up response for user_id=%d: %v", userID, err)
		return false, err
	}
	log.Printf("[RabbitMQ] sign_up response for user_id=%d: %q", userID, resp)
	return resp == "true", nil
}

func (b *Bot) sendRequestPhone(token string, chatID int64) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
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

func (b *Bot) SendMessage(token string, chatID int64, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
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

// Структура для сообщений из RabbitMQ
// Ожидается, что в очереди лежит JSON вида {"chat_id":123, "text":"..."}
type OutboxMessage struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

func (b *Bot) startRabbitListener(token string) {
	queue := "bot_outbox"
	log.Printf("[RabbitMQ] Listening for messages on queue: %s", queue)
	b.rabbitClient.Consume(queue, func(body []byte) {
		var msg OutboxMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			log.Printf("[RabbitMQ] Error decoding outbox message: %v", err)
			return
		}
		log.Printf("[RabbitMQ] Got outbox message: chat_id=%d, text=%q", msg.ChatID, msg.Text)
		b.SendMessage(token, msg.ChatID, msg.Text)
	})
}
