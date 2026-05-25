package protocols

import "context"

// MetricAddress identifies the protocol-specific address for one metric.
type MetricAddress struct {
	Name     string
	Type     string
	Unit     string
	Register uint16
	NodeID   string
	Path     string
	Address  string
	Scale    float64
	Options  map[string]any
}

// MetricValue is the normalized result returned by a protocol reader.
type MetricValue struct {
	Name  string
	Raw   any
	Value any
}

// SourceEndpoint contains the connection details needed by a reader.
type SourceEndpoint struct {
	AgentID string
	Type    string
	Origin  string
	Host    string
	Port    int
	UnitID  byte
	Options map[string]any
}

// Reader reads values from one source endpoint.
type Reader interface {
	Read(ctx context.Context, endpoint SourceEndpoint, metric MetricAddress) (MetricValue, error)
}

// Subscriber streams values from one source endpoint.
type Subscriber interface {
	Subscribe(ctx context.Context, endpoint SourceEndpoint, metrics []MetricAddress, intervalMS int, handler func(MetricValue)) error
}
