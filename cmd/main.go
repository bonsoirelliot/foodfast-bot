package main

import (
	"fmt"
	"foodfast-bot/internal/bot"
	"foodfast-bot/utils"
	"log"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	fmt.Println("Starting bot...")

	utils.InitRedis()

	// слушаем канал редис, чтобы получать сообщения от апи
	utils.StartRequestListener(func(req utils.SendMessageRequest) {
		bot.SendMessage(req.UserID, req.Text)
	})

	bot.StartBot()
}
