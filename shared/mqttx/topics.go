package mqttx

import "strings"

// SplitTopic splits an MQTT topic into path segments.
func SplitTopic(topic string) []string {
	if topic == "" {
		return []string{""}
	}
	return strings.Split(topic, "/")
}

// MetricWildcard returns the common OT.io metric subscription topic.
func MetricWildcard(prefix string) string {
	return strings.TrimRight(prefix, "/") + "/+/metrics/+"
}

// StatusWildcard returns the common OT.io status subscription topic.
func StatusWildcard(prefix string) string {
	return strings.TrimRight(prefix, "/") + "/+/status"
}

// ErrorWildcard returns the common OT.io error subscription topic.
func ErrorWildcard(prefix string) string {
	return strings.TrimRight(prefix, "/") + "/+/errors"
}

// SenseStatusTopic returns the internal Sense status topic used by Lense.
func SenseStatusTopic(prefix string) string {
	return strings.TrimRight(prefix, "/") + "/_sense/status"
}

// MetricTopic returns a concrete OT.io metric topic.
func MetricTopic(prefix string, agentID string, metricName string) string {
	return strings.TrimRight(prefix, "/") + "/" + agentID + "/metrics/" + metricName
}

// StatusTopic returns a concrete OT.io source status topic.
func StatusTopic(prefix string, agentID string) string {
	return strings.TrimRight(prefix, "/") + "/" + agentID + "/status"
}

// ErrorTopic returns a concrete OT.io source error topic.
func ErrorTopic(prefix string, agentID string) string {
	return strings.TrimRight(prefix, "/") + "/" + agentID + "/errors"
}
