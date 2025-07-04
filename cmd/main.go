package main

import (
	"fmt"
	"foodfast-bot/internal/bot"
	"foodfast-bot/internal/domain/user"
	"foodfast-bot/internal/pkg/rabbit"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	fmt.Println("Starting bot...")

	// rabbitURL := os.Getenv("RABBITMQ_URL")
	rabbitClient := rabbit.New("amqp://user:password@rabbitmq:5672/")
	userService := user.New(rabbitClient)
	b := bot.New(userService, rabbitClient)

	b.Start()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop
	fmt.Println("Gracefully stopped")
}
