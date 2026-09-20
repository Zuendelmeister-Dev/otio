package protocols

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var LabProtocolTypes = []string{"modbus-tcp", "modbus-rtu-tcp", "opcua-tcp", "s7", "ethernet-ip", "bacnet-ip", "knxnet-ip", "iec-60870-5-104", "mqtt", "amqp"}

// LabReader delegates a bounded native read to the separately deployed protocol lab.
// Keeping PLC4Go in that service avoids loading every driver into each collector.
type LabReader struct{}

func (LabReader) Read(ctx context.Context, endpoint SourceEndpoint, metric MetricAddress) (MetricValue, error) {
	gateway, _ := endpoint.Options["gatewayURL"].(string)
	if gateway == "" {
		gateway = "http://protocol-lab:8500"
	}
	connection, _ := endpoint.Options["connection"].(string)
	if connection == "" {
		return MetricValue{}, fmt.Errorf("options.connection is required for %s", endpoint.Type)
	}
	address := metric.Address
	if address == "" {
		address = metric.NodeID
	}
	raw, err := json.Marshal(map[string]string{"protocol": strings.TrimPrefix(endpoint.Type, "lab-"), "connection": connection, "address": address})
	if err != nil {
		return MetricValue{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(gateway, "/")+"/api/read", bytes.NewReader(raw))
	if err != nil {
		return MetricValue{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return MetricValue{}, err
	}
	defer response.Body.Close()
	var result struct {
		Value any    `json:"value"`
		Error string `json:"error"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 1024*1024)).Decode(&result); err != nil {
		return MetricValue{}, fmt.Errorf("invalid lab response: %w", err)
	}
	if response.StatusCode != 200 {
		return MetricValue{}, fmt.Errorf("protocol lab HTTP %d: %s", response.StatusCode, result.Error)
	}
	value := result.Value
	if number, ok := value.(float64); ok {
		scale := metric.Scale
		if scale == 0 {
			scale = 1
		}
		value = number * scale
	}
	return MetricValue{Name: metric.Name, Raw: result.Value, Value: value}, nil
}
