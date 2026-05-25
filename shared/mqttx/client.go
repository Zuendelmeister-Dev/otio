package mqttx

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// BrokerConfig contains the common MQTT connection settings used by OT.io modules.
type BrokerConfig struct {
	Host     string
	Port     int
	ClientID string
}

// Hooks contains optional lifecycle callbacks for an MQTT client.
type Hooks struct {
	OnConnect        func(Client)
	OnConnectionLost func(Client, error)
}

// NewClientOptions builds a shared MQTT client configuration.
func NewClientOptions(config BrokerConfig, hooks Hooks) *mqtt.ClientOptions {
	options := mqtt.NewClientOptions()
	options.AddBroker(BrokerURL(config.Host, config.Port))
	options.SetClientID(config.ClientID)
	options.SetAutoReconnect(true)
	options.SetConnectRetry(true)
	if hooks.OnConnect != nil {
		options.OnConnect = hooks.OnConnect
	}
	if hooks.OnConnectionLost != nil {
		options.OnConnectionLost = hooks.OnConnectionLost
	}
	return options
}

// NewClient creates a Paho MQTT client with shared OT.io defaults.
func NewClient(config BrokerConfig, hooks Hooks) Client {
	return mqtt.NewClient(NewClientOptions(config, hooks))
}

// Connect waits for a bounded time and returns the connection error, if any.
func Connect(client Client, timeout time.Duration) error {
	token := client.Connect()
	token.WaitTimeout(timeout)
	return token.Error()
}

// BrokerURL returns a tcp:// MQTT broker URL.
func BrokerURL(host string, port int) string {
	return fmt.Sprintf("tcp://%s:%d", host, port)
}

// IsConnected returns true when the client exists and is connected.
func IsConnected(client Client) bool {
	return client != nil && client.IsConnected()
}
