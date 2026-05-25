package main

import (
	"sync"
	"time"

	"iot-lense-sense/shared/mqttx"
)

type Config struct {
	InstanceID   string       `json:"instanceId"`
	SourceFilter string       `json:"sourceFilter"`
	InputBroker  BrokerConfig `json:"inputBroker"`
	OutputBroker BrokerConfig `json:"outputBroker"`
	SourcePrefix string       `json:"sourcePrefix"`
	TargetPrefix string       `json:"targetPrefix"`
	BufferLimit  int          `json:"bufferLimit"`
}

type BrokerConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	ClientID string `json:"clientId"`
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Component string `json:"component"`
	Message   string `json:"message"`
}

type MetricPoint struct {
	Timestamp string  `json:"timestamp"`
	Epoch     int64   `json:"epoch"`
	Value     float64 `json:"value"`
	Unit      string  `json:"unit"`
}

type MetricMessage struct {
	Timestamp string `json:"timestamp"`
	AgentID   string `json:"agentId"`
	Source    struct {
		Type string `json:"type"`
		Host string `json:"host"`
	} `json:"source"`
	Metric struct {
		Name  string  `json:"name"`
		Value float64 `json:"value"`
		Unit  string  `json:"unit"`
		Type  string  `json:"type"`
	} `json:"metric"`
	Quality struct {
		Status string `json:"status"`
	} `json:"quality"`
}

type State struct {
	sync.Mutex
	Config Config

	InputConnected  bool
	OutputConnected bool
	LastInput       string
	LastOutput      string
	ReceivedCount   int64
	ForwardedCount  int64
	DroppedCount    int64

	InputClient  mqttx.Client
	OutputClient mqttx.Client

	Logs    []LogEntry
	Metrics map[string]map[string][]MetricPoint
	Topics  map[string]bool
	Started time.Time
}

var state = State{
	Metrics: map[string]map[string][]MetricPoint{},
	Topics:  map[string]bool{},
	Started: time.Now(),
}
