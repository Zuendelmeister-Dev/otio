package mqttx

import "testing"

func TestRelativeTopicWithHierarchicalNamespace(t *testing.T) {
	for _, tc := range []struct {
		prefix, topic, agent string
		match                bool
	}{
		{"iot-lense", "iot-lense/machine/metrics/temperature", "machine", true},
		{"factory/line/", "factory/line/_sense/status", "_sense", true},
		{"factory/line", "factory/line/machine/errors", "machine", true},
		{"factory/line", "factory/line-2/machine/status", "", false},
		{"factory/line", "factory/line//status", "", false},
	} {
		parts, match := RelativeTopic(tc.prefix, tc.topic)
		if match != tc.match || match && parts[0] != tc.agent {
			t.Fatalf("%s %s: %v %v", tc.prefix, tc.topic, parts, match)
		}
	}
}
