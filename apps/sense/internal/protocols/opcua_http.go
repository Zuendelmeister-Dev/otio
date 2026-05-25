package protocols

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type OPCUAHTTPReader struct {
	Client http.Client
}

func (reader OPCUAHTTPReader) httpClient() http.Client {
	if reader.Client.Timeout == 0 {
		reader.Client.Timeout = 2 * time.Second
	}
	return reader.Client
}

func (reader OPCUAHTTPReader) Read(ctx context.Context, endpoint SourceEndpoint, metric MetricAddress) (MetricValue, error) {
	nodeID := metric.NodeID
	if nodeID == "" {
		nodeID = metric.Name
	}
	requestURL := fmt.Sprintf("http://%s:%d/read?nodeId=%s", endpoint.Host, endpoint.Port, url.QueryEscape(nodeID))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return MetricValue{}, err
	}
	client := reader.httpClient()
	response, err := client.Do(request)
	if err != nil {
		return MetricValue{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return MetricValue{}, fmt.Errorf("OPC UA read returned HTTP %d", response.StatusCode)
	}
	var payload struct {
		NodeID  string `json:"nodeId"`
		Value   any    `json:"value"`
		Quality string `json:"quality"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return MetricValue{}, err
	}
	return MetricValue{
		Name:  metric.Name,
		Raw:   map[string]any{"nodeId": payload.NodeID, "value": payload.Value, "quality": payload.Quality},
		Value: scaleValue(metric, payload.Value),
	}, nil
}

func (reader OPCUAHTTPReader) Subscribe(ctx context.Context, endpoint SourceEndpoint, metrics []MetricAddress, intervalMS int, handler func(MetricValue)) error {
	metricsByNode := map[string]MetricAddress{}
	nodeIDs := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		nodeID := metric.NodeID
		if nodeID == "" {
			nodeID = metric.Name
		}
		metricsByNode[nodeID] = metric
		nodeIDs = append(nodeIDs, nodeID)
	}
	requestURL := fmt.Sprintf("http://%s:%d/subscribe?nodeIds=%s&intervalMs=%d", endpoint.Host, endpoint.Port, url.QueryEscape(strings.Join(nodeIDs, ",")), intervalMS)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	client := http.Client{Timeout: 0}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("OPC UA subscription returned HTTP %d", response.StatusCode)
	}

	scanner := bufio.NewScanner(response.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payloadRaw := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payloadRaw == "" {
			continue
		}
		var event struct {
			Items []struct {
				NodeID  string `json:"nodeId"`
				Value   any    `json:"value"`
				Quality string `json:"quality"`
			} `json:"items"`
		}
		if err := json.Unmarshal([]byte(payloadRaw), &event); err != nil {
			return err
		}
		for _, item := range event.Items {
			metric, ok := metricsByNode[item.NodeID]
			if !ok {
				continue
			}
			handler(MetricValue{
				Name:  metric.Name,
				Raw:   map[string]any{"nodeId": item.NodeID, "value": item.Value, "quality": item.Quality},
				Value: scaleValue(metric, item.Value),
			})
		}
	}
	return scanner.Err()
}

func scaleValue(metric MetricAddress, rawValue any) any {
	scale := metric.Scale
	if scale == 0 {
		scale = 1
	}
	if value, ok := rawValue.(float64); ok {
		return value * scale
	}
	return rawValue
}
