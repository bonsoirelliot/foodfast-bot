package rabbit

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

type Client struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
}

func New(url string) *Client {
	conn, err := amqp091.Dial(url)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	return &Client{conn: conn, channel: ch}
}

func (c *Client) Publish(queue string, msg interface{}) error {
	body, _ := json.Marshal(msg)
	_, err := c.channel.QueueDeclare(queue, false, false, false, false, nil)
	if err != nil {
		return err
	}
	return c.channel.PublishWithContext(context.Background(), "", queue, false, false,
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
}

func (c *Client) Consume(queue string, handler func([]byte)) error {
	_, err := c.channel.QueueDeclare(queue, false, false, false, false, nil)
	if err != nil {
		return err
	}
	msgs, err := c.channel.Consume(queue, "", true, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for d := range msgs {
			handler(d.Body)
		}
	}()
	return nil
}

func (c *Client) SendRequestAndWaitResponse(requestChannel, responseKey string, payload interface{}, timeout time.Duration) (string, error) {
	err := c.Publish(requestChannel, payload)
	if err != nil {
		return "", err
	}

	_, err = c.channel.QueueDeclare(
		responseKey, // имя очереди
		false,       // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		return "", err
	}

	msgCh := make(chan string, 1)
	done := make(chan struct{})
	go func() {

		c.Consume(responseKey, func(body []byte) {
			msgCh <- string(body)
			close(done)
		})
	}()

	select {
	case msg := <-msgCh:
		return msg, nil
	case <-time.After(timeout):
		return "", errors.New("timeout waiting for response from " + responseKey)
	}
}
