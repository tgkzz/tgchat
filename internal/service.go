package internal

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Service struct {
	Queue   amqp.Queue
	Channel *amqp.Channel
}

type IService interface {
	PublishMessage(ctx context.Context, username, msg string) error
	ConsumeMessage(name string) (<-chan amqp.Delivery, error)
}

func NewService(queue amqp.Queue, channel *amqp.Channel) IService {
	return &Service{
		Queue:   queue,
		Channel: channel,
	}
}

func (s *Service) PublishMessage(ctx context.Context, username, msg string) error {
	return s.Channel.PublishWithContext(ctx, "", s.Queue.Name, false, false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(username + " wrote " + msg),
		})
}

func (s *Service) ConsumeMessage(name string) (<-chan amqp.Delivery, error) {
	msgs, err := s.Channel.Consume(s.Queue.Name, name, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	return msgs, nil
}
