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

		if msg.Text != "" {
			return true
		}

		return false
	}

	dispatcher.AddHandler(handlers.NewCommand("start", h.start))
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

// TODO: method must implement connecting to chat and subscribe to updates in rabbit mq channel
func (h *Handler) start(b *gotgbot.Bot, ctx *ext.Context) error {
	h.userMux.Lock()
	h.users[ctx.EffectiveUser.Id] = ctx.EffectiveUser.Username
	h.userMux.Unlock()

	go h.consume(ctx.EffectiveUser)

	if _, err := ctx.EffectiveMessage.Reply(b, fmt.Sprintf("%s, welcome to the chat", ctx.EffectiveUser.Username), nil); err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

// TODO: method must implement writing message to everyone in the chat
func (h *Handler) write(b *gotgbot.Bot, ctx *ext.Context) error {
	if err := h.Service.PublishMessage(context.Background(), ctx.EffectiveUser.Username, ctx.EffectiveMessage.Text); err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func (h *Handler) consume(user *gotgbot.User) {
	for {
		msgs, err := h.Service.ConsumeMessage(user.Username)
		if err != nil {
			fmt.Println(err)
			continue
		}

		for msg := range msgs {
			h.userMux.Lock()
			for userId := range h.users {
				if userId != user.Id {
					if _, err := h.tgBot.SendMessage(userId, string(msg.Body), nil); err != nil {
						fmt.Println(err)
					}
				}
			}
			h.userMux.Unlock()
		}
	}
}
