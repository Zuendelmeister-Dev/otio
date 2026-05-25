package app

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"iot-lense-sense/shared/mqttx"
)

type AppState struct {
	sync.Mutex
	Config          Config
	ConfigRaw       string
	ConfigPath      string
	HistoryDir      string
	ConfigLoaded    bool
	ConfigValid     bool
	ConfigError     string
	BrokerConnected bool
	LastPublish     string
	PublishCount    int64
	Sources         map[string]SourceStatus
	Logs            []LogEntry
	Metrics         map[string]map[string][]MetricPoint
	Proposals       map[string]Proposal
	MQTTClient      mqttx.Client
	Publisher       EventPublisher
	CancelWorkers   context.CancelFunc
}

var state = &AppState{
	Sources:   map[string]SourceStatus{},
	Logs:      []LogEntry{},
	Metrics:   map[string]map[string][]MetricPoint{},
	Proposals: map[string]Proposal{},
}

func nowISO() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func addLog(level string, component string, message string) {
	entry := LogEntry{Timestamp: nowISO(), Level: level, Component: component, Message: message}
	state.Lock()
	state.Logs = append([]LogEntry{entry}, state.Logs...)
	if len(state.Logs) > 500 {
		state.Logs = state.Logs[:500]
	}
	state.Unlock()
	log.Printf("%s %s %s: %s", entry.Timestamp, level, component, message)
}

func humanAge(ts string) (int64, string) {
	if ts == "" {
		return 0, "n/a"
	}
	parsed, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return 0, "n/a"
	}
	seconds := int64(time.Since(parsed).Seconds())
	if seconds < 0 {
		seconds = 0
	}
	if seconds < 60 {
		return seconds, fmt.Sprintf("%ds ago", seconds)
	}
	if seconds < 3600 {
		return seconds, fmt.Sprintf("%dm %ds ago", seconds/60, seconds%60)
	}
	return seconds, fmt.Sprintf("%dh %dm ago", seconds/3600, (seconds%3600)/60)
}

func snapshotSources() map[string]SourceStatus {
	state.Lock()
	defer state.Unlock()
	out := map[string]SourceStatus{}
	for key, value := range state.Sources {
		seconds, human := humanAge(value.LastRead)
		value.LastReadAgoSeconds = seconds
		value.LastReadAgoHuman = human
		out[key] = value
	}
	return out
}
