package protocols

import "context"

type ModbusTCPReader struct{}

func (reader ModbusTCPReader) Read(ctx context.Context, endpoint SourceEndpoint, metric MetricAddress) (MetricValue, error) {
	raw, err := ReadHoldingRegister(endpoint.Host, endpoint.Port, endpoint.UnitID, metric.Register)
	if err != nil {
		return MetricValue{}, err
	}
	scale := metric.Scale
	if scale == 0 {
		scale = 1
	}
	return MetricValue{
		Name:  metric.Name,
		Raw:   map[string]any{"register": metric.Register, "value": raw},
		Value: float64(raw) * scale,
	}, nil
}
