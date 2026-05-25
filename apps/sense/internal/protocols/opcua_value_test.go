package protocols

import "testing"

func TestScaleValueKeepsStringsAndScalesNumbers(t *testing.T) {
	metric := MetricAddress{Name: "state", Scale: 10}
	if got := scaleValue(metric, "running"); got != "running" {
		t.Fatalf("string value = %#v, want running", got)
	}

	numeric := MetricAddress{Name: "temperature", Scale: 0.1}
	if got := scaleValue(numeric, 235.0); got != 23.5 {
		t.Fatalf("numeric value = %#v, want 23.5", got)
	}
}
