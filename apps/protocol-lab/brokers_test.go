package main

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

func TestBrokerRoundTrips(t *testing.T) {
	for _, protocol := range []string{"mqtt", "amqp"} {
		t.Run(protocol, func(t *testing.T) {
			key := "LAB_TEST_MQTT_URL"
			publishKey := "LAB_MQTT_URL"
			address := "lab/temperature"
			if protocol == "amqp" {
				key = "LAB_TEST_AMQP_URL"
				publishKey = "LAB_AMQP_URL"
				address = "amq.topic/lab.temperature"
			}
			endpoint := os.Getenv(key)
			if endpoint == "" {
				t.Skip("set " + key + " to run the real-broker integration test")
			}
			t.Setenv("LAB_MQTT_URL", "")
			t.Setenv("LAB_AMQP_URL", "")
			t.Setenv(publishKey, endpoint)
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			done := make(chan struct{})
			go func() { defer close(done); publishDemo(ctx, newSimulator()) }()
			defer func() { cancel(); <-done }()
			value, err := readValue(ctx, ReadRequest{protocol, endpoint, address})
			if err != nil {
				t.Fatal(err)
			}
			if value != float64(23.5) {
				t.Fatalf("got %v", value)
			}
		})
	}
}

func TestMQTTPublisherDoesNotStartAMQPPublisher(t *testing.T) {
	mqttListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer mqttListener.Close()
	amqpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer amqpListener.Close()
	observed := make(chan bool, 1)
	go func() {
		conn, err := mqttListener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
		if _, err := packets.ReadPacket(conn); err != nil {
			return
		}
		if err := packets.NewControlPacket(packets.Connack).Write(conn); err != nil {
			return
		}
		packet, err := packets.ReadPacket(conn)
		_, publish := packet.(*packets.PublishPacket)
		observed <- err == nil && publish
	}()
	amqpStarted := make(chan struct{}, 1)
	go func() {
		conn, err := amqpListener.Accept()
		if err == nil {
			conn.Close()
			amqpStarted <- struct{}{}
		}
	}()
	t.Setenv("LAB_MQTT_URL", "tcp://"+mqttListener.Addr().String())
	t.Setenv("LAB_AMQP_URL", "amqp://guest:guest@"+amqpListener.Addr().String()+"/")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	sim := newSimulator()
	publishDemoTick(ctx, sim, "mqtt")
	select {
	case ok := <-observed:
		if !ok {
			t.Fatal("MQTT simulator did not publish")
		}
	case <-ctx.Done():
		t.Fatal("MQTT simulator did not finish")
	}
	select {
	case <-amqpStarted:
		t.Fatal("starting the MQTT simulator also connected to AMQP")
	default:
	}
	if _, ok := sim.publisherStatus()["amqp"]; ok {
		t.Fatal("MQTT-only publication recorded AMQP activity")
	}
}
