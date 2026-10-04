package protocols

import (
	"context"
	"strings"
)

type QualityStatus string

const (
	QualityGood      QualityStatus = "good"
	QualityBad       QualityStatus = "bad"
	QualityUncertain QualityStatus = "uncertain"
	QualityUnknown   QualityStatus = "unknown"
)

// NormalizeQuality accepts the quality classes and OPC UA status-code names.
// Missing or unrecognized upstream quality must never imply a good value.
func NormalizeQuality(status string) QualityStatus {
	status = strings.ToLower(strings.TrimSpace(status))
	switch {
	case strings.HasPrefix(status, "good"):
		return QualityGood
	case strings.HasPrefix(status, "bad"):
		return QualityBad
	case strings.HasPrefix(status, "uncertain"):
		return QualityUncertain
	default:
		return QualityUnknown
	}
}

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
	Name    string
	Raw     any
	Value   any
	Quality QualityStatus
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
