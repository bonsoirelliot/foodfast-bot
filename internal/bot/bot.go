package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"foodfast-bot/utils"
	"os"

	"log"
	"net/http"
)

var telegramToken string

type BotUpdate struct {
	UpdateID int     `json:"update_id"`
	Message  Message `json:"message"`
}

type Message struct {
	MessageID int    `json:"message_id"`
	From      User   `json:"from"`
	Text      string `json:"text"`
	Contact   struct {
		PhoneNumber string `json:"phone_number"`
	} `json:"contact"`
	Chat struct {
		ID int64 `json:"id"`
	} `json:"chat"`
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
}

func StartBot() {
	telegramToken = os.Getenv("TELEGRAM_BOT_TOKEN")

	fmt.Println("token: ", telegramToken)
	offset := 0
	for {
		updates := getUpdates(offset)
		for _, update := range updates {
			offset = update.UpdateID + 1
			handleUpdate(update)
		}
	}
}

func getUpdates(offset int) []BotUpdate {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30", telegramToken, offset)
	resp, err := http.Get(url)
	if err != nil {
		log.Println("getUpdates error:", err)
		return nil
	}
	defer resp.Body.Close()
	var result struct {
		Result []BotUpdate `json:"result"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Result
}

func handleUpdate(update BotUpdate) {
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID
	// firstName := update.Message.From.FirstName // больше не используется

	if update.Message.Text == "/start" {
		//check if user exists in redis
		exists, err := utils.CheckUserExistsPubSub(userID)
		if err != nil {
			SendMessage(chatID, "Ошибка при проверке пользователя. Попробуйте позже.")
			return
		}
		if exists {
			SendMessage(chatID, "Я пока что умею только регистрировать, новый функционал ещё в разработке...")
		} else {
			sendRequestPhone(chatID)
		}
		return
	}

	if update.Message.Contact.PhoneNumber != "" {
		err := utils.RegisterUserPubSub(userID, update.Message.Contact.PhoneNumber, update.Message.From.FirstName)
		if err != nil {
			SendMessage(chatID, "Ошибка при регистрации пользователя. Попробуйте позже.")
			return
		}
		SendMessage(chatID, "Спасибо! Вы зарегистрированы.")
		return
	}
}

func SendMessage(chatID int64, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramToken)
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

func sendRequestPhone(chatID int64) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramToken)
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
