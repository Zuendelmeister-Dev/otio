package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func outputTestConfig() Config {
	return Config{OutputBroker: BrokerConfig{Host: "mqtt", Port: 1883},
		Output:   OutputConfig{Type: "mqtt", TimeoutMs: 100},
		Delivery: DeliveryConfig{MaxRetries: 2, InitialBackoffMs: 2, MaxBackoffMs: 3, QueueLimit: 1}}
}

type scriptedOutput struct {
	mu     sync.Mutex
	errors []error
	ids    []string
	times  []time.Time
}

func (s *scriptedOutput) Ready() bool { return true }
func (s *scriptedOutput) Close()      {}
func (s *scriptedOutput) Write(_ context.Context, message outputMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ids = append(s.ids, message.ID)
	s.times = append(s.times, time.Now())
	if len(s.errors) == 0 {
		return nil
	}
	err := s.errors[0]
	s.errors = s.errors[1:]
	return err
}

func TestDeliveryRetryBoundariesAndBackoff(t *testing.T) {
	retry := &deliveryError{message: "temporary", retryable: true}
	terminal := &deliveryError{message: "permanent"}
	for _, tc := range []struct {
		name          string
		errors        []error
		wantCalls     int
		wantSuccess   bool
		wantUncertain bool
	}{
		{"recover", []error{retry, retry, nil}, 3, true, false},
		{"exhaust", []error{retry, retry, retry, nil}, 3, false, false},
		{"permanent", []error{terminal, nil}, 1, false, false},
		{"unclassified", []error{errors.New("unclassified"), nil}, 1, false, false},
		{"uncertain MQTT", []error{&deliveryError{message: "timeout", uncertain: true}, nil}, 1, false, true},
		{"uncertain HTTP recovered", []error{&deliveryError{message: "timeout", retryable: true, uncertain: true}, nil}, 2, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writer := &scriptedOutput{errors: tc.errors}
			result := deliver(context.Background(), writer, outputTestConfig(), outputMessage{ID: "stable"})
			if len(writer.ids) != tc.wantCalls || (result.err == nil) != tc.wantSuccess || result.uncertain != tc.wantUncertain || result.retries != int64(tc.wantCalls-1) {
				t.Fatalf("calls=%d result=%+v", len(writer.ids), result)
			}
			for i, id := range writer.ids {
				if id != "stable" {
					t.Fatal("retry changed delivery ID")
				}
				if i > 0 && writer.times[i].Sub(writer.times[i-1]) < 2*time.Millisecond {
					t.Fatal("retry skipped backoff")
				}
			}
		})
	}
}

func TestDeliveryCancellationInterruptsBackoff(t *testing.T) {
	config := outputTestConfig()
	config.Delivery.InitialBackoffMs, config.Delivery.MaxBackoffMs = 1000, 1000
	ctx, cancel := context.WithCancel(context.Background())
	writer := &cancelOutput{cancel: cancel}
	start := time.Now()
	result := deliver(ctx, writer, config, outputMessage{})
	if !errors.Is(result.err, context.Canceled) || result.retries != 0 || time.Since(start) > 100*time.Millisecond {
		t.Fatalf("cancellation result: %+v", result)
	}
}

func TestMQTTOutputCancellationDoesNotWaitForPublishTimeout(t *testing.T) {
	started := make(chan struct{})
	client := &routingClient{connected: true, token: routingToken{}, onPublish: func() { close(started) }}
	writer := &mqttOutput{client: client}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- writer.Write(ctx, outputMessage{}) }()
	<-started
	cancel()
	select {
	case err := <-result:
		var failure *deliveryError
		if !errors.As(err, &failure) || !failure.uncertain || failure.retryable {
			t.Fatalf("cancelled in-flight publish: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("MQTT output ignored cancellation")
	}
}

type cancelOutput struct{ cancel context.CancelFunc }

func (c *cancelOutput) Ready() bool { return true }
func (c *cancelOutput) Close()      {}
func (c *cancelOutput) Write(context.Context, outputMessage) error {
	c.cancel()
	return &deliveryError{message: "temporary", retryable: true}
}

func TestHTTPOutputAcceptanceRetryAndIdempotency(t *testing.T) {
	var ids, bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		ids, bodies = append(ids, r.Header.Get("Idempotency-Key")), append(bodies, string(body))
		if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("X-OTIO-Topic") != "target/a/metrics/m" {
			t.Error("incorrect HTTP mapping")
		}
		if len(ids) == 1 {
			w.WriteHeader(503)
		} else {
			w.WriteHeader(202)
		}
	}))
	defer server.Close()
	config := outputTestConfig()
	config.Output.Type, config.Output.URL = "http", server.URL
	writer, err := newHTTPOutput(config.Output)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if writer.Ready() {
		t.Fatal("untested HTTP endpoint must not appear healthy")
	}
	result := deliver(context.Background(), writer, config, outputMessage{ID: "same", Topic: "target/a/metrics/m", Payload: []byte(`{"metric":{"value":"running"}}`)})
	if result.err != nil || result.retries != 1 || !writer.Ready() || len(ids) != 2 || ids[0] != ids[1] || bodies[0] != bodies[1] {
		t.Fatalf("result=%+v ids=%v bodies=%v", result, ids, bodies)
	}
}

func TestHTTPRoutingAccountsForFinalAcceptanceAndRejection(t *testing.T) {
	resetRoutingState(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		code := 401
		if calls == 1 { code = 503 }; if calls == 2 { code = 202 }
		w.WriteHeader(code)
	}))
	defer server.Close()
	state.Config.Output = OutputConfig{Type: "http", URL: server.URL, TimeoutMs: 100}
	state.Config.Delivery.MaxRetries = 2
	writer, err := newHTTPOutput(state.Config.Output)
	if err != nil { t.Fatal(err) }
	state.Output = newDeliveryQueue(context.Background(), writer, state.Config, recordDelivery)
	t.Cleanup(state.Output.Close)
	payload := `{"agentId":"agent","source":{"type":"modbus-tcp"},"metric":{"name":"temperature","value":23.5}}`
	onMetric(nil, routingMessage{payload: payload}); waitRoutingOutput(t)
	state.Lock()
	if state.ReceivedCount != 1 || state.ForwardedCount != 1 || state.DroppedCount != 0 || state.RetryCount != 1 || !writer.Ready() { t.Errorf("successful retry accounting: received=%d forwarded=%d dropped=%d retries=%d", state.ReceivedCount, state.ForwardedCount, state.DroppedCount, state.RetryCount) }
	lastSuccess := state.LastOutput
	state.Unlock()
	onMetric(nil, routingMessage{payload: payload}); waitRoutingOutput(t)
	state.Lock(); defer state.Unlock()
	if state.ReceivedCount != 2 || state.ForwardedCount != 1 || state.DroppedCount != 1 || state.RetryCount != 1 || state.LastOutput != lastSuccess || writer.Ready() { t.Errorf("rejected request changed accepted counter: received=%d forwarded=%d dropped=%d retries=%d", state.ReceivedCount, state.ForwardedCount, state.DroppedCount, state.RetryCount) }
}

func TestHTTPOutputStatusRulesAndNoRedirect(t *testing.T) {
	for _, code := range []int{204, 301, 400, 401, 403, 408, 429, 500, 503} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Location", "/other")
				w.WriteHeader(code)
				_, _ = w.Write([]byte("secret server response"))
			}))
			defer server.Close()
			config := outputTestConfig()
			config.Output.Type, config.Output.URL = "http", server.URL
			writer, _ := newHTTPOutput(config.Output)
			defer writer.Close()
			result := deliver(context.Background(), writer, config, outputMessage{})
			wantCalls := 1
			if code == 408 || code == 429 || code >= 500 {
				wantCalls = 3
			}
			if calls != wantCalls || (result.err == nil) != (code == 204) {
				t.Fatalf("calls=%d result=%+v", calls, result)
			}
			if result.err != nil && strings.Contains(result.err.Error(), "secret") {
				t.Fatal("server body leaked into error")
			}
		})
	}
}

func TestHTTPSOutputTokenAndTrust(t *testing.T) {
	t.Setenv("DISPENSE_TEST_TOKEN", "private-value")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer private-value" {
			t.Error("bearer token missing")
		}
		w.WriteHeader(200)
	}))
	defer server.Close()
	config := outputTestConfig()
	config.Output = OutputConfig{Type: "http", URL: server.URL, BearerTokenEnv: "DISPENSE_TEST_TOKEN", TimeoutMs: 100}
	if err := validateOutputConfig(config); err != nil {
		t.Fatal(err)
	}
	writer, err := newHTTPOutput(config.Output)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := writer.Write(ctx, outputMessage{}); err == nil {
		t.Fatal("untrusted TLS certificate accepted")
	}
	writer.client = server.Client()
	if err := writer.Write(ctx, outputMessage{}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DISPENSE_TEST_TOKEN", "")
	if _, err := newHTTPOutput(config.Output); err == nil || strings.Contains(err.Error(), "private-value") {
		t.Fatal("missing token must fail safely")
	}
}

func TestHTTPOutputTimeoutIsBoundedAndUncertain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	config := outputTestConfig()
	config.Output.Type, config.Output.URL, config.Output.TimeoutMs = "http", server.URL, 50
	config.Delivery.MaxRetries = 1
	writer, _ := newHTTPOutput(config.Output)
	defer writer.Close()
	start := time.Now()
	result := deliver(context.Background(), writer, config, outputMessage{})
	if result.err == nil || !result.uncertain || result.retries != 1 || time.Since(start) > time.Second {
		t.Fatalf("result=%+v elapsed=%v", result, time.Since(start))
	}
}

type blockingOutput struct{ began chan struct{} }

func (b *blockingOutput) Ready() bool { return true }
func (b *blockingOutput) Close()      {}
func (b *blockingOutput) Write(ctx context.Context, _ outputMessage) error {
	b.began <- struct{}{}
	<-ctx.Done()
	return ctx.Err()
}

func TestDeliveryQueueBoundedAndShutdown(t *testing.T) {
	writer := &blockingOutput{began: make(chan struct{}, 1)}
	results := make(chan deliveryResult, 3)
	queue := newDeliveryQueue(context.Background(), writer, outputTestConfig(), func(result deliveryResult) { results <- result })
	if !queue.Submit("a", []byte("first")) {
		t.Fatal("first submit failed")
	}
	<-writer.began
	if !queue.Submit("b", []byte("second")) || queue.Submit("c", nil) {
		t.Fatal("queue bound not enforced")
	}
	queue.Close()
	if queue.Submit("d", nil) || len(results) != 2 {
		t.Fatalf("shutdown lost accounting: results=%d", len(results))
	}
	for i := 0; i < 2; i++ {
		if result := <-results; !errors.Is(result.err, context.Canceled) {
			t.Fatalf("shutdown result=%+v", result)
		}
	}
}

func TestOutputValidationAndLegacyConfig(t *testing.T) {
	t.Setenv("DISPENSE_CONFIG_JSON", `{"outputBroker":{"host":"legacy","port":1884}}`)
	config := loadConfig()
	if config.Output.Type != "mqtt" || config.OutputBroker.Host != "legacy" || config.Delivery.MaxRetries != 0 || validateOutputConfig(config) != nil {
		t.Fatalf("legacy config: %+v", config)
	}
	for _, mutate := range []func(*Config){
		func(c *Config) { c.Output.Type = "unknown" },
		func(c *Config) { c.Output.TimeoutMs = 0 },
		func(c *Config) { c.Delivery.MaxRetries = 6 },
		func(c *Config) { c.Delivery.QueueLimit = 0 },
		func(c *Config) { c.Delivery.MaxBackoffMs = 1 },
		func(c *Config) {
			c.Output = OutputConfig{Type: "http", URL: "http://user:secret@target", TimeoutMs: 100}
		},
		func(c *Config) {
			c.Output = OutputConfig{Type: "http", URL: "http://target", BearerTokenEnv: "SECRET", TimeoutMs: 100}
		},
	} {
		invalid := config
		mutate(&invalid)
		if validateOutputConfig(invalid) == nil {
			t.Fatalf("invalid output accepted: %+v", invalid)
		}
	}
}

func TestOutputAPIDoesNotLeakTokenOrURLQuery(t *testing.T) {
	resetRoutingState(t)
	state.Config.Output = OutputConfig{Type: "http", URL: "https://example.test/telemetry?token=private-query", BearerTokenEnv: "PRIVATE_TOKEN", TimeoutMs: 100}
	t.Setenv("PRIVATE_TOKEN", "private-bearer")
	for _, handler := range []http.HandlerFunc{apiStatus, apiRuntime} {
		response := httptest.NewRecorder()
		handler(response, httptest.NewRequest("GET", "/", nil))
		if strings.Contains(response.Body.String(), "private-") || !strings.Contains(response.Body.String(), "https://example.test/telemetry") {
			t.Fatal("URL query or bearer token exposed in API")
		}
	}
}
