package mqttx

import (
	"context"
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
func (t testToken) Done() <-chan struct{} {
	if !t.complete {
		return nil
	}
	done := make(chan struct{})
	close(done)
	return done
}

type testClient struct {
	Client
	token          mqtt.Token
	connected      bool
	open           bool
	topic          string
	qos            byte
	retained       bool
	payload        any
	published      bool
	closeOnPublish bool
}

func (c *testClient) Connect() mqtt.Token    { return c.token }
func (c *testClient) IsConnected() bool      { return c.connected }
func (c *testClient) IsConnectionOpen() bool { return c.open }
func (c *testClient) Publish(topic string, qos byte, retained bool, payload any) mqtt.Token {
	c.topic, c.qos, c.retained, c.payload, c.published = topic, qos, retained, payload, true
	if c.closeOnPublish {
		c.open = false
	}
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
			client := &testClient{token: tc.token, open: true}
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
	client := &testClient{token: testToken{complete: true}, open: true}
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

func TestReadyDuringReconnect(t *testing.T) {
	if IsReady(nil) || IsReady(&testClient{connected: true}) || !IsReady(&testClient{connected: true, open: true}) {
		t.Fatal("readiness must require an open connection, including during reconnect")
	}
}

func TestPublishDoesNotClaimSuccessWhileReconnecting(t *testing.T) {
	client := &testClient{connected: true, token: testToken{complete: true}}
	if err := Publish(client, "topic", 0, false, "value", time.Second); !errors.Is(err, ErrNotReady) || client.published {
		t.Fatalf("publish during reconnect: error=%v published=%v", err, client.published)
	}
	if err := Publish(nil, "topic", 0, false, nil, time.Second); !errors.Is(err, ErrNotReady) {
		t.Fatalf("nil client: %v", err)
	}
	client = &testClient{open: true, closeOnPublish: true, token: testToken{complete: true}}
	if err := Publish(client, "topic", 0, false, nil, time.Second); !errors.Is(err, ErrConnectionLost) {
		t.Fatalf("connection loss around token: %v", err)
	}
}

func TestPublishContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client := &testClient{open: true, token: testToken{}}
	timer := time.AfterFunc(10*time.Millisecond, cancel)
	defer timer.Stop()
	start := time.Now()
	if err := PublishContext(ctx, client, "topic", 0, false, nil); !errors.Is(err, context.Canceled) || time.Since(start) > time.Second {
		t.Fatalf("cancelled publish: %v", err)
	}
}
