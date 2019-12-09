package rabbit

import (
	"github.com/streadway/amqp"
)

type Consumer struct {
	RabbitChannel   *amqp.Channel
	DeliveryChannel <-chan amqp.Delivery
}
