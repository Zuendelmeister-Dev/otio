package mqttx

import "testing"

func TestTopicHelpers(t *testing.T) {
	tests := map[string]string{
		MetricWildcard("iot-lense/"):                          "iot-lense/+/metrics/+",
		StatusWildcard("iot-lense"):                           "iot-lense/+/status",
		ErrorWildcard("iot-lense"):                            "iot-lense/+/errors",
		SenseStatusTopic("iot-lense"):                         "iot-lense/_sense/status",
		MetricTopic("iot-lense", "machine-01", "temperature"): "iot-lense/machine-01/metrics/temperature",
		StatusTopic("iot-lense", "machine-01"):                "iot-lense/machine-01/status",
		ErrorTopic("iot-lense", "machine-01"):                 "iot-lense/machine-01/errors",
	}
	for got, want := range tests {
		if got != want {
			t.Fatalf("topic helper = %s, want %s", got, want)
		}
	}
}

func TestSplitTopic(t *testing.T) {
	parts := SplitTopic("iot-lense/machine-01/metrics/temperature")
	if len(parts) != 4 || parts[1] != "machine-01" || parts[3] != "temperature" {
		t.Fatalf("unexpected parts: %#v", parts)
	}
}
