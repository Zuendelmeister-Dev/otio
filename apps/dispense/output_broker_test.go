package main

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"iot-lense-sense/shared/mqttx"
)

// This local wire fixture accepts real Paho MQTT connections, records QoS 0
// publishes and can break the connection. No Docker or external broker needed.
type wireBroker struct {
	listener    net.Listener
	online      atomic.Bool
	mu          sync.Mutex
	connections []net.Conn
	messages    chan string
	done        chan struct{}
}

func newWireBroker(t *testing.T) *wireBroker {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	b := &wireBroker{listener: listener, messages: make(chan string, 8), done: make(chan struct{})}
	b.online.Store(true)
	go func() {
		defer close(b.done)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			b.mu.Lock()
			b.connections = append(b.connections, conn)
			b.mu.Unlock()
			go b.serve(conn)
		}
	}()
	t.Cleanup(func() { listener.Close(); <-b.done; b.setOnline(false) })
	return b
}

func (b *wireBroker) setOnline(value bool) {
	b.online.Store(value)
	if !value {
		b.mu.Lock()
		defer b.mu.Unlock()
		for _, conn := range b.connections {
			conn.Close()
		}
	}
}

func mqttPacket(conn net.Conn) (byte, []byte, error) {
	var first [1]byte
	if _, err := io.ReadFull(conn, first[:]); err != nil {
		return 0, nil, err
	}
	size, multiplier := 0, 1
	for i := 0; i < 4; i++ {
		var next [1]byte
		if _, err := io.ReadFull(conn, next[:]); err != nil {
			return 0, nil, err
		}
		size += int(next[0]&127) * multiplier
		if next[0]&128 == 0 {
			if size > 65536 {
				return 0, nil, io.ErrShortBuffer
			}
			body := make([]byte, size)
			_, err := io.ReadFull(conn, body)
			return first[0], body, err
		}
		multiplier *= 128
	}
	return 0, nil, io.ErrUnexpectedEOF
}

func (b *wireBroker) serve(conn net.Conn) {
	defer conn.Close()
	if !b.online.Load() {
		return
	}
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	kind, _, err := mqttPacket(conn)
	if err != nil || kind>>4 != 1 || !b.online.Load() {
		return
	}
	if _, err := conn.Write([]byte{0x20, 2, 0, 0}); err != nil {
		return
	}
	for {
		kind, body, err := mqttPacket(conn)
		if err != nil {
			return
		}
		switch kind >> 4 {
		case 3:
			if len(body) < 2 {
				return
			}
			length := int(binary.BigEndian.Uint16(body[:2]))
			if len(body) < 2+length {
				return
			}
			b.messages <- string(body[2+length:])
		case 12:
			if _, err := conn.Write([]byte{0xd0, 0}); err != nil {
				return
			}
		case 14:
			return
		}
	}
}

func waitBrokerState(t *testing.T, client mqttx.Client, ready bool) {
	t.Helper()
	until := time.Now().Add(5 * time.Second)
	for time.Now().Before(until) {
		if mqttx.IsReady(client) == ready {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("MQTT open state did not become %t", ready)
}

func TestMQTTOutputActualDisconnectAndReconnect(t *testing.T) {
	broker := newWireBroker(t)
	host, port, _ := net.SplitHostPort(broker.listener.Addr().String())
	number, _ := strconv.Atoi(port)
	options := mqttx.NewClientOptions(mqttx.BrokerConfig{Host: host, Port: number, ClientID: "dispense-wire-test"}, mqttx.Hooks{})
	options.SetConnectTimeout(100 * time.Millisecond).SetConnectRetryInterval(50 * time.Millisecond).SetMaxReconnectInterval(100 * time.Millisecond)
	client := mqtt.NewClient(options)
	if err := mqttx.Connect(client, time.Second); err != nil {
		t.Fatal(err)
	}
	writer := &mqttOutput{client: client}
	defer writer.Close()
	config := outputTestConfig()
	first := deliver(context.Background(), writer, config, outputMessage{ID: "first", Topic: "target/a/metrics/m", Payload: []byte("first")})
	if first.err != nil {
		t.Fatal(first.err)
	}
	select {
	case got := <-broker.messages:
		if got != "first" {
			t.Fatal(got)
		}
	case <-time.After(time.Second):
		t.Fatal("broker did not receive initial message")
	}
	broker.setOnline(false)
	waitBrokerState(t, client, false)
	// Paho's logical connected state remains true during auto reconnect.
	if !client.IsConnected() {
		t.Fatal("fixture did not exercise Paho reconnect state")
	}
	missing := deliver(context.Background(), writer, config, outputMessage{ID: "during-outage", Topic: "target/a/metrics/m", Payload: []byte("must-not-count")})
	if missing.err == nil || missing.retries != 2 || missing.uncertain {
		t.Fatalf("outage result: %+v", missing)
	}
	broker.setOnline(true)
	waitBrokerState(t, client, true)
	recovered := deliver(context.Background(), writer, config, outputMessage{ID: "after", Topic: "target/a/metrics/m", Payload: []byte("after")})
	if recovered.err != nil {
		t.Fatal(recovered.err)
	}
	select {
	case got := <-broker.messages:
		if got != "after" {
			t.Fatalf("unexpected replay or payload: %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("broker did not receive recovered message")
	}
}
