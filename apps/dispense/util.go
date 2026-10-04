package main

import (
	"encoding/json"
	"os"
	"strconv"
	"time"
)

func getenv(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func getenvInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func addLog(level, component, message string) {
	state.Lock()
	defer state.Unlock()
	state.Logs = append(state.Logs, LogEntry{Timestamp: nowISO(), Level: level, Component: component, Message: message})
	if len(state.Logs) > 500 {
		state.Logs = state.Logs[len(state.Logs)-500:]
	}
}

func loadConfig() Config {
	config := Config{
		InstanceID:   getenv("DISPENSE_INSTANCE_ID", "iot-dispense-01"),
		SourceFilter: getenv("DISPENSE_SOURCE_FILTER", "modbus-tcp"),
		SourcePrefix: getenv("DISPENSE_SOURCE_PREFIX", "iot-lense"),
		TargetPrefix: getenv("DISPENSE_TARGET_PREFIX", "dispense/out"),
		BufferLimit:  getenvInt("DISPENSE_BUFFER_LIMIT", 1500),
		Output: OutputConfig{
			Type:           getenv("DISPENSE_OUTPUT_TYPE", "mqtt"),
			URL:            getenv("DISPENSE_HTTP_URL", ""),
			BearerTokenEnv: getenv("DISPENSE_HTTP_TOKEN_ENV", ""),
			TimeoutMs:      getenvInt("DISPENSE_OUTPUT_TIMEOUT_MS", 2000),
		},
		Delivery: DeliveryConfig{
			MaxRetries:       getenvInt("DISPENSE_MAX_RETRIES", 0),
			InitialBackoffMs: getenvInt("DISPENSE_RETRY_BACKOFF_MS", 200),
			MaxBackoffMs:     getenvInt("DISPENSE_MAX_BACKOFF_MS", 2000),
			QueueLimit:       getenvInt("DISPENSE_QUEUE_LIMIT", 128),
		},
		InputBroker: BrokerConfig{
			Host:     getenv("DISPENSE_INPUT_BROKER_HOST", "mqtt"),
			Port:     getenvInt("DISPENSE_INPUT_BROKER_PORT", 1883),
			ClientID: getenv("DISPENSE_INPUT_CLIENT_ID", getenv("DISPENSE_INSTANCE_ID", "iot-dispense-01")+"-in"),
		},
		OutputBroker: BrokerConfig{
			Host:     getenv("DISPENSE_OUTPUT_BROKER_HOST", "dispense-broker"),
			Port:     getenvInt("DISPENSE_OUTPUT_BROKER_PORT", 1883),
			ClientID: getenv("DISPENSE_OUTPUT_CLIENT_ID", getenv("DISPENSE_INSTANCE_ID", "iot-dispense-01")+"-out"),
		},
	}
	if raw := getenv("DISPENSE_CONFIG_JSON", ""); raw != "" {
		candidate := config
		if err := json.Unmarshal([]byte(raw), &candidate); err != nil {
			addLog("ERROR", "config", "Invalid DISPENSE_CONFIG_JSON: "+err.Error())
		} else {
			config = candidate
		}
	}
	if config.BufferLimit < 1 {
		config.BufferLimit = 1500
	}
	return config
}
