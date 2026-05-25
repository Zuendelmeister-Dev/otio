package main

import (
	"encoding/json"
	"time"

	"iot-lense-sense/shared/mqttx"
)

func connectInput(config Config) mqttx.Client {
	client := mqttx.NewClient(mqttx.BrokerConfig{
		Host:     config.InputBroker.Host,
		Port:     config.InputBroker.Port,
		ClientID: config.InputBroker.ClientID,
	}, mqttx.Hooks{
		OnConnect: func(client mqttx.Client) {
			state.Lock()
			state.InputConnected = true
			state.Unlock()
			topic := mqttx.MetricWildcard(config.SourcePrefix)
			client.Subscribe(topic, 0, onMetric)
			addLog("INFO", "input-mqtt", "Subscribed to "+topic)
		},
		OnConnectionLost: func(client mqttx.Client, err error) {
			state.Lock()
			state.InputConnected = false
			state.Unlock()
			addLog("WARN", "input-mqtt", "Connection lost: "+err.Error())
		},
	})

	if err := mqttx.Connect(client, 8*time.Second); err != nil {
		addLog("ERROR", "input-mqtt", err.Error())
	}
	return client
}

func connectOutput(config Config) mqttx.Client {
	client := mqttx.NewClient(mqttx.BrokerConfig{
		Host:     config.OutputBroker.Host,
		Port:     config.OutputBroker.Port,
		ClientID: config.OutputBroker.ClientID,
	}, mqttx.Hooks{
		OnConnect: func(client mqttx.Client) {
			state.Lock()
			state.OutputConnected = true
			state.Unlock()
			addLog("INFO", "output-mqtt", "Connected to target broker")
		},
		OnConnectionLost: func(client mqttx.Client, err error) {
			state.Lock()
			state.OutputConnected = false
			state.Unlock()
			addLog("WARN", "output-mqtt", "Connection lost: "+err.Error())
		},
	})

	if err := mqttx.Connect(client, 8*time.Second); err != nil {
		addLog("ERROR", "output-mqtt", err.Error())
	}
	return client
}

func onMetric(client mqttx.Client, msg mqttx.Message) {
	state.Lock()
	config := state.Config
	output := state.OutputClient
	state.ReceivedCount++
	state.LastInput = nowISO()
	state.Topics[msg.Topic()] = true
	state.Unlock()

	var metric MetricMessage
	if err := json.Unmarshal(msg.Payload(), &metric); err != nil {
		addLog("WARN", "router", "Dropped invalid JSON from "+msg.Topic())
		return
	}
	if config.SourceFilter != "" && metric.Source.Type != config.SourceFilter {
		return
	}

	metricName := metric.Metric.Name
	if metricName == "" {
		parts := mqttx.SplitTopic(msg.Topic())
		if len(parts) > 0 {
			metricName = parts[len(parts)-1]
		}
	}
	targetTopic := mqttx.MetricTopic(config.TargetPrefix, metric.AgentID, metricName)

	state.Lock()
	if _, ok := state.Metrics[metric.AgentID]; !ok {
		state.Metrics[metric.AgentID] = map[string][]MetricPoint{}
	}
	state.Metrics[metric.AgentID][metricName] = append(state.Metrics[metric.AgentID][metricName], MetricPoint{
		Timestamp: metric.Timestamp,
		Epoch:     time.Now().UnixMilli(),
		Value:     metric.Metric.Value,
		Unit:      metric.Metric.Unit,
	})
	if len(state.Metrics[metric.AgentID][metricName]) > config.BufferLimit {
		state.Metrics[metric.AgentID][metricName] = state.Metrics[metric.AgentID][metricName][len(state.Metrics[metric.AgentID][metricName])-config.BufferLimit:]
	}
	state.Unlock()

	if !mqttx.IsConnected(output) {
		state.Lock()
		state.DroppedCount++
		state.Unlock()
		addLog("WARN", "router", "Target broker not connected. Dropped "+targetTopic)
		return
	}

	if err := mqttx.Publish(output, targetTopic, 0, false, msg.Payload(), 2*time.Second); err != nil {
		state.Lock()
		state.DroppedCount++
		state.Unlock()
		addLog("ERROR", "router", err.Error())
		return
	}

	state.Lock()
	state.ForwardedCount++
	state.LastOutput = nowISO()
	state.Unlock()
}
