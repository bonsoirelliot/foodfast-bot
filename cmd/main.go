package main

import (
	"fmt"
	"foodfast-bot/internal/bot"
	"foodfast-bot/internal/domain/user"
	"foodfast-bot/internal/pkg/redis"
	"log"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	fmt.Println("Starting bot...")

	redisClient := redis.New()
	userService := user.New(redisClient)
	bot := bot.New(userService, redisClient)

	bot.Start()
}
