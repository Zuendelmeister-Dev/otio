package main

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestTCPDemoReads(t *testing.T) {
	for _, protocol := range []string{"s7", "mbus-tcp"} {
		t.Run(protocol, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
			defer cancel()
			l, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer l.Close()
			go serveTCPDemo(ctx, l, newSimulator(), protocol)
			connection := fmt.Sprintf("%s://%s", protocol, l.Addr())
			address := "1"
			if protocol == "s7" {
				connection += "?remote-rack=0&remote-slot=1&controller-type=S7_1200"
				address = "%DB1:0:REAL"
			}
			value, err := readValue(ctx, ReadRequest{protocol, connection, address})
			if err != nil {
				t.Fatal(err)
			}
			if fmt.Sprint(value) != "23.5" {
				t.Fatalf("unexpected value %v", value)
			}
		})
	}
}
func TestMBusRejectsCorruptionAndUnsupportedRecords(t *testing.T) {
	frame := mbusDemoFrame(23.5)
	tail := frame[4:]
	if _, err := decodeMBusTemperature(tail, 2); err == nil {
		t.Fatal("accepted wrong address")
	}
	tail[17] ^= 1
	if _, err := decodeMBusTemperature(tail, 1); err == nil {
		t.Fatal("accepted checksum corruption")
	}
	frame = mbusDemoFrame(23.5)
	tail = frame[4:]
	tail[16] = 0x13
	tail[len(tail)-2] -= 0x47
	if _, err := decodeMBusTemperature(tail, 1); err == nil {
		t.Fatal("guessed unsupported unit")
	}
}
func TestSimulationRuntimeDefaultsOff(t *testing.T) {
	runtime := newSimulationRuntime(context.Background(), newSimulator())
	defer runtime.stopAll()
	if len(runtime.names()) != 0 {
		t.Fatal("simulators must be opt-in")
	}
	if err := runtime.set("profibus", true); err == nil {
		t.Fatal("unsupported simulator accepted")
	}
	if err := runtime.set("s7", false); err != nil {
		t.Fatal(err)
	}
}
