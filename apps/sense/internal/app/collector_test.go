package app

import (
	"io"
	"net"
	"testing"
	"time"

	"iot-lense-sense/sense/internal/protocols"
)

type unreadyRestartPublisher struct{ disconnected bool }

func (*unreadyRestartPublisher) Connected() bool               { return false }
func (*unreadyRestartPublisher) PublishJSON(string, any) error { return nil }
func (p *unreadyRestartPublisher) Disconnect(uint)             { p.disconnected = true }

func TestRestartDisconnectsUnreadyPreviousPublisher(t *testing.T) {
	// Accept the replacement's real MQTT handshake so this exercises the full
	// restart without relying on an external broker or waiting for a timeout.
	t.Setenv("OTIO_MQTT_USERNAME", "")
	t.Setenv("OTIO_MQTT_PASSWORD", "")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	broker := make(chan net.Conn, 1)
	brokerError := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			brokerError <- err
			return
		}
		broker <- conn
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		var header [2]byte
		if _, err = io.ReadFull(conn, header[:]); err == nil && header[0] == 0x10 && header[1] < 128 {
			_, err = io.CopyN(io.Discard, conn, int64(header[1]))
			if err == nil {
				_, err = conn.Write([]byte{0x20, 2, 0, 0})
			}
		}
		brokerError <- err
	}()
	previousState := state
	oldPublisher := &unreadyRestartPublisher{}
	oldWorkersCanceled := false
	state = &AppState{
		Config:        Config{Broker: BrokerConfig{Host: "127.0.0.1", Port: listener.Addr().(*net.TCPAddr).Port, ClientID: "sense-restart-test"}},
		Publisher:     oldPublisher,
		CancelWorkers: func() { oldWorkersCanceled = true },
	}
	t.Cleanup(func() {
		state.CancelWorkers()
		if state.Publisher != nil {
			state.Publisher.Disconnect(0)
		}
		select {
		case conn := <-broker:
			conn.Close()
		default:
		}
		state = previousState
	})
	restartWorkers()
	if !oldWorkersCanceled || !oldPublisher.disconnected {
		t.Fatalf("old worker/client survived restart: workers canceled=%t, publisher disconnected=%t", oldWorkersCanceled, oldPublisher.disconnected)
	}
	if state.Publisher == oldPublisher || state.MQTTClient == nil {
		t.Fatal("restart did not install a replacement publisher")
	}
	if err := <-brokerError; err != nil {
		t.Fatal(err)
	}
}

func TestPayloadBuilders(t *testing.T) {
	source := SourceConfig{AgentID: "machine-01", Type: "modbus-tcp", Host: "host", Port: 5020, UnitID: 1}
	metric := MetricConfig{Name: "temperature", Register: 0, Scale: 0.1, Unit: "°C", Type: "gauge"}
	payload := metricPayload(source, metric, protocols.MetricValue{Raw: 235, Value: 23.5, Quality: protocols.QualityGood})
	if payload["agentId"] != "machine-01" {
		t.Fatalf("unexpected agent id")
	}
	if payload["metric"].(map[string]any)["value"] != 23.5 {
		t.Fatalf("unexpected metric value")
	}
	if statusPayload(source, true, true, "ok")["connected"] != true {
		t.Fatalf("unexpected status payload")
	}
	if errorPayload(source, "boom")["message"] != "boom" {
		t.Fatalf("unexpected error payload")
	}
}

func TestMetricPayloadDoesNotPromoteBadOrUnknownQuality(t *testing.T) {
	for _, quality := range []protocols.QualityStatus{protocols.QualityGood, protocols.QualityBad, protocols.QualityUncertain, protocols.QualityUnknown, ""} {
		payload := metricPayload(SourceConfig{}, MetricConfig{}, protocols.MetricValue{Value: 42.0, Quality: quality})
		want := string(protocols.NormalizeQuality(string(quality)))
		if got := payload["quality"].(map[string]any)["status"]; got != want {
			t.Fatalf("normalized quality=%v, want %s", got, want)
		}
	}
}

func TestRememberMetric(t *testing.T) {
	state.Lock()
	state.Metrics = map[string]map[string][]MetricPoint{}
	state.Unlock()
	rememberMetric("machine-01", MetricConfig{Name: "temperature", Unit: "°C"}, 23.5)
	state.Lock()
	points := state.Metrics["machine-01"]["temperature"]
	state.Unlock()
	if len(points) != 1 || points[0].Value != 23.5 {
		t.Fatalf("unexpected points: %#v", points)
	}
}
