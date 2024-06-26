package main

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"os"
	"os/signal"
	"tgchat/internal"
)

func main() {
	// Connecting to RabbitMQ
	rconn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		fmt.Println("Failed to connect to RabbitMQ:", err)
		return
	}
	defer rconn.Close()

	// Declaring queue
	rch, err := rconn.Channel()
	if err != nil {
		fmt.Println("Failed to open a channel:", err)
		return
	}
	defer func() {
		if err := rch.Close(); err != nil {
			fmt.Println("Failed to close RabbitMQ channel:", err)
		}
	}()

	q, err := rch.QueueDeclare(
		"message_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		fmt.Println("Failed to declare a queue:", err)
		return
	}

	// Starting Telegram bot
	tgBot, err := gotgbot.NewBot(os.Getenv("TOKEN"), nil)
	if err != nil {
		fmt.Println("Failed to create new bot:", err)
		return
	}

	s := internal.NewService(q, rch)
	h := internal.NewHandler(tgBot, s)

	if err := h.RunTgBot(); err != nil {
		fmt.Println("Failed to run bot:", err)
		return
	}

	// Graceful shutdown handling
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	<-ch

	fmt.Println("...project is shutting down")
}
