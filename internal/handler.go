package internal

import (
	"context"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"log"
	"sync"
	"time"
)

type Handler struct {
	tgBot   *gotgbot.Bot
	Service IService
	users   map[int64]string
	userMux sync.Mutex
}

func NewHandler(tgbot *gotgbot.Bot, service IService) *Handler {
	return &Handler{tgBot: tgbot, Service: service, users: make(map[int64]string)}
}

func (h *Handler) RunTgBot() error {
	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			log.Println("an error occurred while handling update:", err.Error())
			return ext.DispatcherActionNoop
		},
		MaxRoutines: ext.DefaultMaxRoutines,
	})

	filter := func(msg *gotgbot.Message) bool {
		return msg.Text != ""
	}

	dispatcher.AddHandler(handlers.NewCommand("start", h.start))
	dispatcher.AddHandler(handlers.NewCommand("inspect", h.inspect))
	dispatcher.AddHandler(handlers.NewMessage(filter, h.write))

	updater := ext.NewUpdater(dispatcher, nil)
	if err := updater.StartPolling(h.tgBot, &ext.PollingOpts{
		DropPendingUpdates: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			RequestOpts: &gotgbot.RequestOpts{
				Timeout: time.Second * 10,
			},
		},
	}); err != nil {
		return err
	}

	log.Print("bot has been started \n", "username", h.tgBot.User.Username)

	go updater.Idle()

	return nil
}

func (h *Handler) start(b *gotgbot.Bot, ctx *ext.Context) error {
	h.userMux.Lock()
	h.users[ctx.EffectiveUser.Id] = ctx.EffectiveUser.Username
	h.userMux.Unlock()

	go h.consume(ctx.EffectiveUser.Id)

	if _, err := ctx.EffectiveMessage.Reply(b, fmt.Sprintf("%s, welcome to the chat", ctx.EffectiveUser.Username), nil); err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func (h *Handler) write(b *gotgbot.Bot, ctx *ext.Context) error {
	if err := h.Service.PublishMessage(context.Background(), ctx.EffectiveUser.Username, ctx.EffectiveMessage.Text); err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func (h *Handler) inspect(b *gotgbot.Bot, ctx *ext.Context) error {
	queueName, cons, msgs, err := h.Service.InspectQueue()
	if err != nil {
		fmt.Println(err)
		return err
	}

	if _, err := ctx.EffectiveMessage.Reply(b, fmt.Sprintf("queueName: %s\nconsumers: %d\nmessages: %d\n", queueName, cons, msgs), nil); err != nil {
		return err
	}

	return nil
}

func (h *Handler) consume(me int64) {
	for {
		msgs, err := h.Service.ConsumeMessage("telegram_bot")
		if err != nil {
			fmt.Println(err)
			continue
		}

		for msg := range msgs {
			h.userMux.Lock()
			for userId := range h.users {
				if me != userId {
					if _, err := h.tgBot.SendMessage(userId, string(msg.Body), nil); err != nil {
						fmt.Println(err)
					}
				}
			}
			h.userMux.Unlock()

			if err := msg.Ack(false); err != nil {
				fmt.Println(err)
				return
			}
		}
	}
}
