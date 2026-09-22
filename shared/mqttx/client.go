package mqttx

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// ErrTimeout means the operation did not complete within the requested wait.
// The underlying MQTT operation may still complete later.
var ErrTimeout = errors.New("MQTT operation timed out")

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
	// Kubernetes Secrets can supply broker credentials without embedding them
	// in the configuration UI or generated example files.
	options.SetUsername(os.Getenv("OTIO_MQTT_USERNAME"))
	options.SetPassword(os.Getenv("OTIO_MQTT_PASSWORD"))
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
	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("connect: %w", ErrTimeout)
	}
	return token.Error()
}

// BrokerURL returns a tcp:// MQTT broker URL.
func BrokerURL(host string, port int) string {
	return "tcp://" + net.JoinHostPort(host, strconv.Itoa(port))
}

// IsConnected returns true when the client exists and is connected.
func IsConnected(client Client) bool {
	return client != nil && client.IsConnected()
}
