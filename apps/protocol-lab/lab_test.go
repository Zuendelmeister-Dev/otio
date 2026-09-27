package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCatalogAndRequestValidation(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range catalog {
		if seen[p.ID] {
			t.Fatal("duplicate protocol", p.ID)
		}
		seen[p.ID] = true
		err := validateRead(ReadRequest{p.ID, p.Connection, p.Address})
		if (p.Connection != "") != (err == nil) {
			t.Fatalf("catalog default %s: %v", p.ID, err)
		}
	}
	for _, r := range []ReadRequest{{"s7", "tcp://localhost:102", "x"}, {"unknown", "tcp://localhost:1", "x"}, {"mqtt", "tcp://", "x"}, {"amqp", "amqp://localhost", ""}} {
		if validateRead(r) == nil {
			t.Fatalf("accepted invalid request %+v", r)
		}
	}
}
func TestAPIValidationAndRead(t *testing.T) {
	called := 0
	h := handler(newSimulator(), func(ctx context.Context, r ReadRequest) (any, error) { called++; return "running", nil })
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{"GET", "/api/catalog", "", 200},
		{"GET", "/api/status", "", 200},
		{"POST", "/api/read", `{"protocol":"mqtt","connection":"tcp://localhost:1883","address":"test"}`, 200},
		{"POST", "/api/read", `{"protocol":"ethercat"}`, 422},
		{"POST", "/api/read", `{"extra":true}`, 400},
		{"POST", "/api/read", `{} {}`, 400},
		{"PUT", "/api/simulator", `{"temperature":-1,"mode":"constant"}`, 400},
		{"PUT", "/api/simulator", `{"temperature":42,"mode":"constant","running":true}`, 200},
		{"GET", "/api/read", "", 405},
		{"GET", "/", "", 307},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
		if w.Code != tc.status {
			t.Fatalf("%s %s: %d %s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}
	if called != 1 {
		t.Fatalf("invalid requests reached reader: %d", called)
	}
}

func TestReadDeadlineRetainsSlotForStalledDriver(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	h := handler(newSimulator(), func(context.Context, ReadRequest) (any, error) { <-release; return 1, nil })
	for i := 0; i < 8; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		req := httptest.NewRequest("POST", "/api/read", strings.NewReader(`{"protocol":"s7","connection":"s7://localhost:102","address":"%DB1:0:REAL"}`)).WithContext(ctx)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		cancel()
		if w.Code != 504 {
			t.Fatalf("deadline returned %d", w.Code)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/api/read", strings.NewReader(`{"protocol":"s7","connection":"s7://localhost:102","address":"%DB1:0:REAL"}`)))
	if w.Code != 429 {
		t.Fatalf("stalled workers must retain their slots: %d", w.Code)
	}
}
func TestModbusNativeRoundTrip(t *testing.T) {
	for _, rtu := range []bool{false, true} {
		t.Run(fmt.Sprint(rtu), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			sim := newSimulator()
			l, err := listenModbus(ctx, sim, 0, rtu)
			if err != nil {
				t.Fatal(err)
			}
			defer l.Close()
			scheme, protocol := "modbus-tcp", "modbus-tcp"
			if rtu {
				scheme, protocol = "modbus-rtu:tcp", "modbus-rtu-tcp"
			}
			value, err := readValue(ctx, ReadRequest{protocol, fmt.Sprintf("%s://127.0.0.1:%d?default-unit-identifier=1", scheme, l.Addr().(*net.TCPAddr).Port), "holding-register:1:UINT"})
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := json.Marshal(value)
			if string(raw) != "2350" {
				t.Fatalf("got %s (%T)", raw, value)
			}
		})
	}
}
func TestOPCUANativeRoundTrip(t *testing.T) {
	// Reserve a free port; the OPC UA server API accepts a port rather than a listener.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	srv, err := startOPCUA(ctx, newSimulator(), "127.0.0.1", port)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	for _, tc := range []struct {
		node string
		want any
	}{{"Temperature", 23.5}, {"Running", true}} {
		value, err := readValue(ctx, ReadRequest{"opcua-tcp", fmt.Sprintf("opc.tcp://127.0.0.1:%d", port), "ns=1;s=" + tc.node})
		if err != nil {
			t.Fatal(err)
		}
		if value != tc.want {
			t.Fatalf("%s=%v", tc.node, value)
		}
	}
}
func TestSimulatorValidationAndExceptions(t *testing.T) {
	sim := newSimulator()
	if sim.update(simValues{Mode: "bad"}) == nil {
		t.Fatal("accepted invalid mode")
	}
	for _, pdu := range [][]byte{{6, 0, 0, 0, 1}, {3, 0, 2, 0, 1}, {3, 0, 0, 0, 0}, {3}} {
		r := modbusResponse(sim, 1, pdu)
		if len(r) != 2 || r[0]&0x80 == 0 {
			t.Fatalf("missing exception for %v", pdu)
		}
	}
	if crc16([]byte{1, 3, 0, 0, 0, 1}) != 0x0a84 {
		t.Fatal("incorrect RTU CRC")
	}
}
