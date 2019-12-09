package rabbit

import (
	"fmt"
	"sync"

	"github.com/streadway/amqp"
)

var connectOnce sync.Once
var createPubChannelOnce sync.Once
var disconnectOnce sync.Once

var conn *amqp.Connection
var publishChannel *amqp.Channel
var consumerMap = make(map[string]*Consumer)

// Connect creates a rabbit connection if one hasn't already been created
func Connect() error {
	var err error

	connectOnce.Do(func() {
		fmt.Println("connecting to rabbit")
		conn, err = amqp.Dial("amqp://localhost:5672")
	})

	return err
}

// Disconnect ensures all kernel buffers on server and client have been flushed. Closes all underlying io,
// channels, notifiers and consumers followed by the connection
func Disconnect() error {
	if conn == nil {
		return nil
	}

	var err error

	disconnectOnce.Do(func() {
		fmt.Println("Disconnecting from rabbit")

		if publishChannel != nil {
			fmt.Println("Closing publish channel...")
			publishChannel.Close()
		}

		fmt.Println("Gracefully stopping consumers")
		for queueName, consumer := range consumerMap {
			fmt.Println("closing", queueName, "delivery chan")
			consumer.RabbitChannel.Cancel(queueName, false)

			fmt.Println("closing", queueName, "rabbit channel")
			consumer.RabbitChannel.Close()
		}

		err = conn.Close()
	})

	return err
}

// CreateQueue creates queue of name `queueName`
func CreateQueue(queueName string) error {
	err := getPublishChannel()

	if err != nil {
		return err
	}

	_, err = publishChannel.QueueDeclare(queueName, true, false, false, false, nil)

	return err
}

// Publish enqueues the message provided to the queue name provided
func Publish(queueName string, message string) error {
	err := getPublishChannel()

	if err != nil {
		return err
	}

	publishing := amqp.Publishing{Body: []byte(message)}
	err = publishChannel.Publish("", queueName, false, false, publishing)

	return err
}

// Consume consumes messages from the queue name provided and hands them to the handler function provided
func Consume(queueName string, handler func(message string, ack func() error, nack func(bool) error)) error {
	var err error

	err = Connect()

	if err != nil {
		return err
	}

	channel, err := conn.Channel()
	if err != nil {
		return err
	}

	// channel.Qos(50, 0, false)

	deliveryChannel, err := channel.Consume(queueName, queueName, false, false, false, false, nil)
	if err != nil {
		return err
	}

	consumerMap[queueName] = &Consumer{
		RabbitChannel:   channel,
		DeliveryChannel: deliveryChannel,
	}

	go func() {
		for delivery := range deliveryChannel {
			message := string(delivery.Body)

			ack := createAck(delivery.DeliveryTag, channel)
			nack := func(requeue bool) error {
				fmt.Println(delivery.DeliveryTag)
				return channel.Nack(delivery.DeliveryTag, false, requeue)
			}

			go handler(message, ack, nack)
		}
	}()

	return err
}

func createAck(deliveryTag uint64, rabbitChannel *amqp.Channel) func() error {
	return func() error {
		return rabbitChannel.Ack(deliveryTag, false)
	}
}

func getPublishChannel() error {
	var err error
	err = Connect()

	if err != nil {
		return err
	}

	createPubChannelOnce.Do(func() {
		fmt.Println("creating publish channel")

		publishChannel, err = conn.Channel()
	})

	return err
}
