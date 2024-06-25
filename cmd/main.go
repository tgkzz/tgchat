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

	//connecting to rabbit
	rconn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rconn.Close()

	// declaring queue
	rch, err := rconn.Channel()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rch.Close()

	q, err := rch.QueueDeclare("message_queue", true, false, false, false, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	// starting tgbot
	tgBot, err := gotgbot.NewBot(os.Getenv("TOKEN"), nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	s := internal.NewService(q, rch)

	h := internal.NewHandler(tgBot, s)

	if err := h.RunTgBot(); err != nil {
		fmt.Println(err)
		return
	}

	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt)

	<-ch

	fmt.Println("....project is shutting down")
}
