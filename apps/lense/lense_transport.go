package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"iot-lense-sense/shared/mqttx"
)

func nowISO() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func startMQTT() {
	prefix := getenv("MQTT_TOPIC_PREFIX", "iot-lense")
	port, err := strconv.Atoi(getenv("MQTT_PORT", "1883"))
	if err != nil {
		port = 1883
	}

	client := mqttx.NewClient(mqttx.BrokerConfig{
		Host:     getenv("MQTT_BROKER", "mqtt"),
		Port:     port,
		ClientID: "iot-lense",
	}, mqttx.Hooks{
		OnConnect: func(client mqttx.Client) {
			state.Lock()
			state.BrokerConnected = true
			state.Unlock()
			client.Subscribe(mqttx.MetricWildcard(prefix), 0, onMessage)
			client.Subscribe(mqttx.StatusWildcard(prefix), 0, onMessage)
			client.Subscribe(mqttx.ErrorWildcard(prefix), 0, onMessage)
			client.Subscribe(mqttx.SenseStatusTopic(prefix), 0, onMessage)
			log.Println("connected to MQTT broker")
		},
		OnConnectionLost: func(client mqttx.Client, err error) {
			state.Lock()
			state.BrokerConnected = false
			state.Unlock()
		},
	})

	for {
		if err := mqttx.Connect(client, 30*time.Second); err == nil {
			break
		} else {
			log.Println(err)
			time.Sleep(3 * time.Second)
		}
	}
	_ = os.Stdout
}

func onMessage(client mqttx.Client, msg mqttx.Message) {
	topic := msg.Topic()
	state.Lock()
	state.LastMessage = nowISO()
	state.MessageCount++
	state.Topics[topic] = true
	state.Unlock()
	parts := mqttx.SplitTopic(topic)
	raw := string(msg.Payload())
	if len(parts) >= 4 && parts[len(parts)-2] == "metrics" {
		storeMetric(topic, raw)
		return
	}
	if len(parts) >= 3 && parts[len(parts)-1] == "status" && parts[1] == "_sense" {
		storeSenseComponentStatus(raw)
		return
	}
	if len(parts) >= 3 && parts[len(parts)-1] == "status" && parts[1] != "_sense" {
		storeStatus(raw)
		return
	}
	if len(parts) >= 3 && parts[len(parts)-1] == "errors" {
		storeError(raw)
		return
	}
}

func splitTopic(topic string) []string {
	return mqttx.SplitTopic(topic)
}

func storeMetric(topic string, raw string) {
	var m MetricMessage
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return
	}
	var numericValue any
	var textValue any
	switch value := m.Metric.Value.(type) {
	case float64:
		numericValue = value
	case string:
		textValue = value
	default:
		textValue = fmt.Sprintf("%v", value)
	}
	_, _ = db.Exec(`INSERT INTO metric_events (ts,agent_id,topic,metric_name,metric_value,metric_text,metric_unit,metric_type,source_type,source_host,source_address,quality_status,payload) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb)`, m.Timestamp, m.AgentID, topic, m.Metric.Name, numericValue, textValue, m.Metric.Unit, m.Metric.Type, m.Source.Type, m.Source.Host, m.Source.Address, m.Quality.Status, raw)
}
func storeStatus(raw string) {
	var s StatusMessage
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return
	}
	_, _ = db.Exec(`INSERT INTO agent_status (agent_id,ts,connected,healthy,source_type,source_host,payload) VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb) ON CONFLICT (agent_id) DO UPDATE SET ts=EXCLUDED.ts, connected=EXCLUDED.connected, healthy=EXCLUDED.healthy, source_type=EXCLUDED.source_type, source_host=EXCLUDED.source_host, payload=EXCLUDED.payload`, s.AgentID, s.Timestamp, s.Connected, s.Healthy, s.Source.Type, s.Source.Host, raw)
}
func storeError(raw string) {
	var e ErrorMessage
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		return
	}
	_, _ = db.Exec(`INSERT INTO error_events (ts,agent_id,severity,message,payload) VALUES ($1,$2,$3,$4,$5::jsonb)`, e.Timestamp, e.AgentID, e.Severity, e.Message, raw)
}
