package main

import (
	"context"
	"os"
	"testing"
	"time"
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
