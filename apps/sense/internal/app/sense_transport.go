package app

import (
	"strings"
	"time"

	"iot-lense-sense/shared/mqttx"
)

// EventPublisher is the outbound transport seam for Sense.
// MQTT is the first implementation. Kafka, NATS or HTTP publishers can be added
// behind this interface without changing protocol readers.
type EventPublisher interface {
	Connected() bool
	PublishJSON(topic string, payload any) error
	Disconnect(quiesce uint)
}

type mqttEventPublisher struct {
	client mqttx.Client
}

func (publisher mqttEventPublisher) Connected() bool {
	return mqttx.IsConnected(publisher.client)
}

func (publisher mqttEventPublisher) PublishJSON(topic string, payload any) error {
	retained := strings.HasSuffix(topic, "/status")
	return mqttx.PublishJSON(publisher.client, topic, 0, retained, payload, 2*time.Second)
}

func (publisher mqttEventPublisher) Disconnect(quiesce uint) {
	if publisher.client != nil {
		publisher.client.Disconnect(quiesce)
	}
}

func connectTransport(config Config) {
	client := mqttx.NewClient(mqttx.BrokerConfig{
		Host:     config.Broker.Host,
		Port:     config.Broker.Port,
		ClientID: config.Broker.ClientID,
	}, mqttx.Hooks{
		OnConnect: func(client mqttx.Client) {
			state.Lock()
			state.BrokerConnected = true
			state.Unlock()
			addLog("INFO", "mqtt", "Connected to broker")
		},
		OnConnectionLost: func(client mqttx.Client, err error) {
			state.Lock()
			state.BrokerConnected = false
			state.Unlock()
			addLog("WARN", "mqtt", "Disconnected from broker: "+err.Error())
		},
	})

	if err := mqttx.Connect(client, 5*time.Second); err != nil {
		addLog("ERROR", "mqtt", err.Error())
	}

	state.Lock()
	state.MQTTClient = client
	state.Publisher = mqttEventPublisher{client: client}
	state.Unlock()
}

func publishJSON(topic string, payload any) {
	state.Lock()
	publisher := state.Publisher
	state.Unlock()

	if publisher == nil || !publisher.Connected() {
		return
	}
	if err := publisher.PublishJSON(topic, payload); err != nil {
		addLog("ERROR", "transport", err.Error())
		return
	}

	state.Lock()
	state.LastPublish = nowISO()
	state.PublishCount++
	state.Unlock()
}
