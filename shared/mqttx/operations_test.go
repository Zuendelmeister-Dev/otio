package mqttx

import (
	"errors"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type testToken struct {
	mqtt.Token
	complete bool
	err      error
}

func (t testToken) WaitTimeout(time.Duration) bool { return t.complete }
func (t testToken) Error() error                   { return t.err }

type testClient struct {
	Client
	token     mqtt.Token
	connected bool
	topic     string
	qos       byte
	retained  bool
	payload   any
	published bool
}

func (c *testClient) Connect() mqtt.Token { return c.token }
func (c *testClient) IsConnected() bool   { return c.connected }
func (c *testClient) Publish(topic string, qos byte, retained bool, payload any) mqtt.Token {
	c.topic, c.qos, c.retained, c.payload, c.published = topic, qos, retained, payload, true
	return c.token
}

func TestOperationOutcomes(t *testing.T) {
	brokerErr := errors.New("broker rejected operation")
	for _, tc := range []struct {
		name  string
		token testToken
		want  error
	}{
		{"success", testToken{complete: true}, nil},
		{"broker error", testToken{complete: true, err: brokerErr}, brokerErr},
		{"timeout", testToken{}, ErrTimeout},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &testClient{token: tc.token}
			if err := Connect(client, time.Millisecond); !errors.Is(err, tc.want) {
				t.Fatalf("Connect: %v, want %v", err, tc.want)
			}
			if err := Publish(client, "topic", 1, true, "value", time.Millisecond); !errors.Is(err, tc.want) {
				t.Fatalf("Publish: %v, want %v", err, tc.want)
			}
			if client.topic != "topic" || client.qos != 1 || !client.retained || client.payload != "value" {
				t.Fatalf("publish arguments: %+v", client)
			}
		})
	}
}

func TestPublishJSON(t *testing.T) {
	client := &testClient{token: testToken{complete: true}}
	if err := PublishJSON(client, "json", 0, false, map[string]int{"value": 42}, time.Second); err != nil {
		t.Fatal(err)
	}
	if string(client.payload.([]byte)) != `{"value":42}` {
		t.Fatalf("unexpected payload %s", client.payload)
	}
	client.published = false
	if err := PublishJSON(client, "json", 0, false, make(chan int), time.Second); err == nil || client.published {
		t.Fatal("invalid JSON must fail before publishing")
	}
}

func TestClientLifecycle(t *testing.T) {
	if IsConnected(nil) || IsConnected(&testClient{}) || !IsConnected(&testClient{connected: true}) {
		t.Fatal("incorrect connection status")
	}
	connected, lost := false, false
	options := NewClientOptions(BrokerConfig{Host: "::1", Port: 1883}, Hooks{
		OnConnect:        func(Client) { connected = true },
		OnConnectionLost: func(Client, error) { lost = true },
	})
	options.OnConnect(nil)
	options.OnConnectionLost(nil, nil)
	if !connected || !lost || options.Servers[0].String() != "tcp://[::1]:1883" {
		t.Fatal("hooks or IPv6 broker URL incorrect")
	}
	if NewClient(BrokerConfig{Host: "localhost", Port: 1883}, Hooks{}) == nil {
		t.Fatal("nil client")
	}
}
