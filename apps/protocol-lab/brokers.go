package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	amqp "github.com/rabbitmq/amqp091-go"
)

func mqttClient(connection string) mqtt.Client {
	options := mqtt.NewClientOptions().AddBroker(connection).SetClientID(fmt.Sprintf("otio-lab-%d", time.Now().UnixNano())).SetAutoReconnect(false).SetConnectTimeout(4 * time.Second)
	return mqtt.NewClient(options)
}
func waitMQTT(ctx context.Context, token mqtt.Token) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		return token.Error()
	}
}
func sampleMQTT(ctx context.Context, r ReadRequest) (any, error) {
	client := mqttClient(r.Connection)
	if err := waitMQTT(ctx, client.Connect()); err != nil {
		return nil, err
	}
	defer client.Disconnect(0)
	messages := make(chan []byte, 1)
	if err := waitMQTT(ctx, client.Subscribe(r.Address, 0, func(_ mqtt.Client, m mqtt.Message) {
		select {
		case messages <- append([]byte(nil), m.Payload()...):
		default:
		}
	})); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case raw := <-messages:
		return jsonValue(raw)
	}
}
func dialAMQP(ctx context.Context, connection string) (*amqp.Connection, error) {
	return amqp.DialConfig(connection, amqp.Config{Dial: func(network, address string) (net.Conn, error) {
		conn, err := (&net.Dialer{}).DialContext(ctx, network, address)
		if err == nil {
			if deadline, ok := ctx.Deadline(); ok {
				_ = conn.SetDeadline(deadline)
			}
		}
		return conn, err
	}})
}
func sampleAMQP(ctx context.Context, r ReadRequest) (any, error) {
	exchange, key, ok := strings.Cut(r.Address, "/")
	if !ok || exchange == "" || key == "" {
		return nil, errors.New("AMQP address must be exchange/routing-key")
	}
	conn, err := dialAMQP(ctx, r.Connection)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	defer ch.Close()
	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		return nil, err
	}
	if err = ch.QueueBind(q.Name, key, exchange, false, nil); err != nil {
		return nil, err
	}
	deliveries, err := ch.Consume(q.Name, "", true, true, false, false, nil)
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case message, ok := <-deliveries:
		if !ok {
			return nil, errors.New("AMQP subscription closed")
		}
		return jsonValue(message.Body)
	}
}

func publishDemo(ctx context.Context, sim *simulator) { publishDemoProtocol(ctx, sim, "") }
func publishDemoProtocol(ctx context.Context, sim *simulator, protocol string) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			func() {
				raw, _ := json.Marshal(sim.snapshot().Temperature)
				if broker := os.Getenv("LAB_MQTT_URL"); broker != "" && (protocol == "" || protocol == "mqtt") {
					tick, cancel := context.WithTimeout(ctx, 4*time.Second)
					client := mqttClient(broker)
					if err := waitMQTT(tick, client.Connect()); err == nil {
						err = waitMQTT(tick, client.Publish("lab/temperature", 0, true, raw))
						sim.recordPublish("mqtt", err)
					} else {
						sim.recordPublish("mqtt", err)
					}
					client.Disconnect(0)
					cancel()
				}
				if broker := os.Getenv("LAB_AMQP_URL"); broker != "" {
					tick, cancel := context.WithTimeout(ctx, 4*time.Second)
					defer cancel()
					conn, err := dialAMQP(tick, broker)
					if err != nil {
						sim.recordPublish("amqp", err)
						return
					}
					defer conn.Close()
					ch, err := conn.Channel()
					if err != nil {
						sim.recordPublish("amqp", err)
						return
					}
					defer ch.Close()
					err = ch.PublishWithContext(tick, "amq.topic", "lab.temperature", false, false, amqp.Publishing{ContentType: "application/json", Body: raw})
					sim.recordPublish("amqp", err)
				}
			}()
		}
	}
}
