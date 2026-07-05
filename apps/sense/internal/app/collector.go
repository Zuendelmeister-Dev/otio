package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"iot-lense-sense/sense/internal/protocols"
	"iot-lense-sense/shared/mqttx"
)

type protocolRegistry interface {
	Reader(sourceType string) (protocols.Reader, error)
	Subscriber(sourceType string) (protocols.Subscriber, error)
}

var sourceProtocols protocolRegistry = protocols.NewRegistry()

func sourceEndpoint(source SourceConfig) protocols.SourceEndpoint {
	return protocols.SourceEndpoint{
		AgentID: source.AgentID,
		Type:    source.Type,
		Origin:  source.Origin,
		Host:    source.Host,
		Port:    source.Port,
		UnitID:  source.UnitID,
		Options: source.Options,
	}
}

func metricAddressConfig(source SourceConfig, metric MetricConfig) protocols.MetricAddress {
	return protocols.MetricAddress{
		Name:     metric.Name,
		Type:     metric.Type,
		Unit:     metric.Unit,
		Register: metric.Register,
		NodeID:   metric.NodeID,
		Path:     metric.Path,
		Address:  metric.Address,
		Scale:    metric.Scale,
		Options:  metric.Options,
	}
}

func metricAddress(source SourceConfig, metric MetricConfig) string {
	if metric.Address != "" {
		return metric.Address
	}
	if metric.Path != "" {
		return metric.Path
	}
	if source.Type == "opcua" {
		if metric.NodeID != "" {
			return metric.NodeID
		}
		return metric.Name
	}
	return fmt.Sprintf("holding-register:%d", metric.Register)
}

func metricPayload(source SourceConfig, metric MetricConfig, raw any, value any) map[string]any {
	return map[string]any{
		"schemaVersion": "1.0",
		"timestamp":     nowISO(),
		"agentId":       source.AgentID,
		"source":        map[string]any{"type": source.Type, "host": source.Host, "port": source.Port, "unitId": source.UnitID, "address": metricAddress(source, metric)},
		"metric":        map[string]any{"name": metric.Name, "value": value, "unit": metric.Unit, "type": metric.Type},
		"quality":       map[string]any{"status": "good"},
		"raw":           raw,
	}
}

func statusPayload(source SourceConfig, connected bool, healthy bool, message string) map[string]any {
	return map[string]any{"schemaVersion": "1.0", "timestamp": nowISO(), "agentId": source.AgentID, "connected": connected, "healthy": healthy, "message": message, "source": map[string]any{"type": source.Type, "host": source.Host, "port": source.Port, "unitId": source.UnitID}}
}

func errorPayload(source SourceConfig, message string) map[string]any {
	return map[string]any{"schemaVersion": "1.0", "timestamp": nowISO(), "agentId": source.AgentID, "severity": "error", "message": message, "source": map[string]any{"type": source.Type, "host": source.Host, "port": source.Port, "unitId": source.UnitID}}
}

func sourceReadMode(source SourceConfig) string {
	if source.ReadMode == "" {
		return "poll"
	}
	return source.ReadMode
}

func sourceSubscriptionInterval(source SourceConfig, config Config) time.Duration {
	intervalMS := source.SubscriptionIntervalMS
	if intervalMS <= 0 {
		intervalMS = config.PollIntervalMS
	}
	if intervalMS <= 0 {
		intervalMS = 1000
	}
	if intervalMS < 250 {
		intervalMS = 250
	}
	return time.Duration(intervalMS) * time.Millisecond
}

func scaleOPCUAValue(metric MetricConfig, rawValue any) any {
	scale := metric.Scale
	if scale == 0 {
		scale = 1
	}
	if value, ok := rawValue.(float64); ok {
		return value * scale
	}
	return rawValue
}

func metricConfigByName(source SourceConfig) map[string]MetricConfig {
	result := map[string]MetricConfig{}
	for _, metric := range source.Metrics {
		result[metric.Name] = metric
	}
	return result
}

func publishSourceFailure(prefix string, source SourceConfig, message string) {
	now := nowISO()
	state.Lock()
	previous, hadPrevious := state.Sources[source.AgentID]
	changed := !hadPrevious || previous.Connected || previous.Healthy || previous.Message != message
	state.Sources[source.AgentID] = SourceStatus{Connected: false, Healthy: false, LastRead: now, Message: message}
	state.Unlock()

	if changed {
		addLog("ERROR", source.AgentID, message)
		publishJSON(mqttx.ErrorTopic(prefix, source.AgentID), errorPayload(source, message))
	}
	publishJSON(mqttx.StatusTopic(prefix, source.AgentID), statusPayload(source, false, false, message))
}

func rememberMetric(agentID string, metric MetricConfig, value float64) {
	now := time.Now().UTC()
	point := MetricPoint{Timestamp: now.Format(time.RFC3339Nano), Epoch: now.Unix(), Value: value, Unit: metric.Unit}
	state.Lock()
	if _, ok := state.Metrics[agentID]; !ok {
		state.Metrics[agentID] = map[string][]MetricPoint{}
	}
	points := append(state.Metrics[agentID][metric.Name], point)
	cutoff := now.Add(-5 * time.Minute).Unix()
	start := 0
	for ; start < len(points); start++ {
		if points[start].Epoch >= cutoff {
			break
		}
	}
	state.Metrics[agentID][metric.Name] = points[start:]
	state.Unlock()
}

func readMetricValue(source SourceConfig, metric MetricConfig) (any, any, error) {
	reader, err := sourceProtocols.Reader(source.Type)
	if err != nil {
		return nil, nil, err
	}
	value, err := reader.Read(context.Background(), sourceEndpoint(source), metricAddressConfig(source, metric))
	if err != nil {
		return nil, nil, err
	}
	return value.Raw, value.Value, nil
}

func publishMetric(prefix string, source SourceConfig, metric MetricConfig, raw any, value any) {
	publishJSON(mqttx.MetricTopic(prefix, source.AgentID, metric.Name), metricPayload(source, metric, raw, value))
	if numericValue, ok := value.(float64); ok {
		rememberMetric(source.AgentID, metric, numericValue)
	}
}

func markSourceHealthy(prefix string, source SourceConfig, message string) {
	now := nowISO()
	state.Lock()
	previous, hadPrevious := state.Sources[source.AgentID]
	recovered := !hadPrevious || !previous.Connected || !previous.Healthy
	state.Sources[source.AgentID] = SourceStatus{Connected: true, Healthy: true, LastRead: now, Message: message}
	state.Unlock()

	if recovered {
		addLog("INFO", source.AgentID, "Source recovered: "+message)
	}
	publishJSON(mqttx.StatusTopic(prefix, source.AgentID), statusPayload(source, true, true, "Connected"))
}

func pollSource(ctx context.Context, config Config, source SourceConfig) {
	prefix := config.Broker.TopicPrefix
	interval := time.Duration(config.PollIntervalMS) * time.Millisecond
	if interval <= 0 {
		interval = time.Second
	}
	reader, err := sourceProtocols.Reader(source.Type)
	if err != nil {
		publishSourceFailure(prefix, source, err.Error())
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var metricNames []string
		var readError error
		for _, metric := range source.Metrics {
			value, err := reader.Read(ctx, sourceEndpoint(source), metricAddressConfig(source, metric))
			if err != nil {
				readError = err
				break
			}
			metricNames = append(metricNames, metric.Name)
			publishMetric(prefix, source, metric, value.Raw, value.Value)
		}

		if readError != nil {
			publishSourceFailure(prefix, source, readError.Error())
		} else {
			markSourceHealthy(prefix, source, "Read cycle completed for "+strings.Join(metricNames, ", "))
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

func subscribeSource(ctx context.Context, config Config, source SourceConfig) {
	prefix := config.Broker.TopicPrefix
	subscriber, err := sourceProtocols.Subscriber(source.Type)
	if err != nil {
		publishSourceFailure(prefix, source, err.Error())
		return
	}

	metrics := make([]protocols.MetricAddress, 0, len(source.Metrics))
	for _, metric := range source.Metrics {
		metrics = append(metrics, metricAddressConfig(source, metric))
	}
	metricsByName := metricConfigByName(source)
	interval := sourceSubscriptionInterval(source, config)
	retryDelay := time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var received []string
		err := subscriber.Subscribe(ctx, sourceEndpoint(source), metrics, int(interval/time.Millisecond), func(value protocols.MetricValue) {
			metric, ok := metricsByName[value.Name]
			if !ok {
				return
			}
			received = append(received, metric.Name)
			publishMetric(prefix, source, metric, value.Raw, value.Value)

			// A subscription can be a long-running stream. In that case Subscribe
			// does not return after the first successful value, so the source must be
			// marked healthy while values are received. Without this, Lense can show
			// a red source/edge although metrics are already flowing through MQTT.
			markSourceHealthy(prefix, source, "Subscription update received for "+metric.Name)
		})
		if len(received) > 0 {
			markSourceHealthy(prefix, source, "Subscription update received for "+strings.Join(received, ", "))
		}
		if err != nil && ctx.Err() == nil {
			publishSourceFailure(prefix, source, err.Error())
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(retryDelay):
		}
	}
}

func collectSource(ctx context.Context, config Config, source SourceConfig) {
	if sourceReadMode(source) == "subscription" {
		subscribeSource(ctx, config, source)
		return
	}
	pollSource(ctx, config, source)
}

func restartWorkers() {
	state.Lock()
	if state.CancelWorkers != nil {
		state.CancelWorkers()
	}
	config := state.Config
	if state.Publisher != nil && state.Publisher.Connected() {
		state.Publisher.Disconnect(250)
	}
	ctx, cancel := context.WithCancel(context.Background())
	state.CancelWorkers = cancel
	state.Sources = map[string]SourceStatus{}
	state.Unlock()
	connectTransport(config)
	for _, source := range config.Sources {
		state.Lock()
		state.Sources[source.AgentID] = SourceStatus{Connected: false, Healthy: false, Message: "Starting"}
		state.Unlock()
		go collectSource(ctx, config, source)
		addLog("INFO", "collector", "Started "+sourceReadMode(source)+" collection for "+source.AgentID)
	}
	go publishSenseStatus(ctx, config)
}

var restartWorkersAfterConfigApply = restartWorkers

func publishSenseStatus(ctx context.Context, config Config) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			state.Lock()
			connected := state.BrokerConnected
			state.Unlock()
			publishJSON(mqttx.SenseStatusTopic(config.Broker.TopicPrefix), map[string]any{"schemaVersion": "1.0", "timestamp": nowISO(), "senseId": config.Broker.ClientID, "brokerConnected": connected, "sources": snapshotSources()})
		}
	}
}

func testRead(source SourceConfig) map[string]any {
	result := map[string]any{"agentId": source.AgentID, "type": source.Type, "ok": true, "values": map[string]any{}}
	for _, metric := range source.Metrics {
		_, value, err := readMetricValue(source, metric)
		if err != nil {
			result["ok"] = false
			result["error"] = err.Error()
			return result
		}
		result["values"].(map[string]any)[metric.Name] = value
	}
	return result
}
