package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sync"
	"time"

	"iot-lense-sense/shared/mqttx"
)

// outputWriter implementations receive the original JSON and a routing topic.
// Write must honour the deadline and return a deliveryError for retry decisions.
// A nil error acknowledges only the transport's documented acceptance boundary.
type outputWriter interface {
	Write(context.Context, outputMessage) error
	Ready() bool
	Close()
}

type outputMessage struct {
	ID      string
	Topic   string
	Payload []byte
}

type deliveryError struct {
	message   string
	retryable bool
	uncertain bool
}

func (e *deliveryError) Error() string { return e.message }

func validateOutputConfig(config Config) error {
	if config.Output.TimeoutMs < 50 || config.Output.TimeoutMs > 30000 {
		return errors.New("output.timeoutMs must be between 50 and 30000")
	}
	if config.Delivery.MaxRetries < 0 || config.Delivery.MaxRetries > 5 {
		return errors.New("delivery.maxRetries must be between 0 and 5")
	}
	if config.Delivery.QueueLimit < 1 || config.Delivery.QueueLimit > 10000 {
		return errors.New("delivery.queueLimit must be between 1 and 10000")
	}
	if config.Delivery.InitialBackoffMs < 1 || config.Delivery.MaxBackoffMs < config.Delivery.InitialBackoffMs || config.Delivery.MaxBackoffMs > 30000 {
		return errors.New("delivery backoff must be positive, ordered and at most 30000 ms")
	}
	switch config.Output.Type {
	case "mqtt":
		if config.OutputBroker.Host == "" || config.OutputBroker.Port < 1 || config.OutputBroker.Port > 65535 {
			return errors.New("outputBroker requires a host and valid port")
		}
	case "http":
		u, err := url.Parse(config.Output.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
			return errors.New("output.url must be an HTTP/HTTPS URL without user credentials or a fragment")
		}
		if config.Output.BearerTokenEnv != "" {
			if !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(config.Output.BearerTokenEnv) {
				return errors.New("output.bearerTokenEnv must name an environment variable")
			}
			if u.Scheme != "https" {
				return errors.New("authenticated HTTP output requires HTTPS")
			}
		}
	default:
		return errors.New("output.type must be mqtt or http")
	}
	return nil
}

type mqttOutput struct{ client mqttx.Client }

func (m *mqttOutput) Ready() bool { return mqttx.IsReady(m.client) }
func (m *mqttOutput) Close() {
	if m.client != nil {
		m.client.Disconnect(0)
	}
}
func (m *mqttOutput) Write(ctx context.Context, message outputMessage) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if !m.Ready() {
		return &deliveryError{message: "MQTT output connection is not open", retryable: true}
	}
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("MQTT output requires a deadline")
	}
	if err := mqttx.PublishContext(ctx, m.client, message.Topic, 0, false, message.Payload); err != nil {
		if errors.Is(err, mqttx.ErrNotReady) {
			return &deliveryError{message: "MQTT output connection is not open", retryable: true}
		}
		// Once Publish has been called, delivery may complete after a timeout.
		// Retrying would risk a duplicate; only pre-publish disconnects are retried.
		return &deliveryError{message: "MQTT output publish failed or timed out; delivery is uncertain", uncertain: true}
	}
	return nil
}

type httpOutput struct {
	url    string
	token  string
	client *http.Client
	mu     sync.RWMutex
	ready  bool
}

func newHTTPOutput(config OutputConfig) (*httpOutput, error) {
	token := ""
	if config.BearerTokenEnv != "" {
		token = os.Getenv(config.BearerTokenEnv)
		if token == "" || bytes.ContainsAny([]byte(token), "\r\n") {
			return nil, errors.New("HTTP output bearer token is missing or invalid in the configured environment variable")
		}
	}
	return &httpOutput{
		url: config.URL, token: token,
		client: &http.Client{
			Transport: http.DefaultTransport.(*http.Transport).Clone(),
			Timeout:       time.Duration(config.TimeoutMs) * time.Millisecond,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

func (h *httpOutput) Ready() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.ready
}
func (h *httpOutput) Close() { h.client.CloseIdleConnections() }
func (h *httpOutput) setReady(value bool) {
	h.mu.Lock()
	h.ready = value
	h.mu.Unlock()
}
func (h *httpOutput) Write(ctx context.Context, message outputMessage) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(message.Payload))
	if err != nil {
		return &deliveryError{message: "HTTP output request is invalid"}
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-OTIO-Topic", message.Topic)
	request.Header.Set("Idempotency-Key", message.ID)
	// Keep retry counts in this worker. net/http otherwise treats the key as
	// permission to replay POSTs automatically on a broken reused connection.
	request.GetBody = nil
	if h.token != "" {
		request.Header.Set("Authorization", "Bearer "+h.token)
	}
	response, err := h.client.Do(request)
	if err != nil {
		h.setReady(false)
		// Never log the URL, token, request body or server's error response.
		return &deliveryError{message: "HTTP output request failed or timed out; delivery is uncertain", retryable: true, uncertain: true}
	}
	defer response.Body.Close()
	// Do not allow a large/streaming response to monopolize the delivery worker.
	_, _ = io.CopyN(io.Discard, response.Body, 4096)
	accepted := response.StatusCode >= 200 && response.StatusCode < 300
	h.setReady(accepted)
	if accepted {
		return nil
	}
	return &deliveryError{
		message:   fmt.Sprintf("HTTP output rejected message with status %d", response.StatusCode),
		retryable: response.StatusCode == 408 || response.StatusCode == 429 || (response.StatusCode >= 500 && response.StatusCode <= 599),
	}
}

type deliveryResult struct {
	message   outputMessage
	err       error
	retries   int64
	uncertain bool
}

func deliver(ctx context.Context, writer outputWriter, config Config, message outputMessage) deliveryResult {
	result := deliveryResult{message: message}
	backoff := time.Duration(config.Delivery.InitialBackoffMs) * time.Millisecond
	maximum := time.Duration(config.Delivery.MaxBackoffMs) * time.Millisecond
	for attempt := 0; ; attempt++ {
		if ctx.Err() != nil {
			result.err = ctx.Err()
			return result
		}
		if attempt > 0 {
			result.retries++
		}
		attemptCtx, cancel := context.WithTimeout(ctx, time.Duration(config.Output.TimeoutMs)*time.Millisecond)
		result.err = writer.Write(attemptCtx, message)
		cancel()
		var failure *deliveryError
		if errors.As(result.err, &failure) && failure.uncertain {
			result.uncertain = true
		}
		if result.err == nil || !errors.As(result.err, &failure) || !failure.retryable || attempt >= config.Delivery.MaxRetries {
			return result
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			result.err = ctx.Err()
			return result
		case <-timer.C:
		}
		backoff = min(backoff*2, maximum)
	}
}

// A single bounded worker preserves submission order while keeping the MQTT
// receive callback short. This queue is intentionally process-local, not an outbox.
type deliveryQueue struct {
	writer outputWriter
	queue  chan outputMessage
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	mu     sync.Mutex
	closed bool
	result func(deliveryResult)
}

func newDeliveryQueue(ctx context.Context, writer outputWriter, config Config, result func(deliveryResult)) *deliveryQueue {
	ctx, cancel := context.WithCancel(ctx)
	q := &deliveryQueue{writer: writer, queue: make(chan outputMessage, config.Delivery.QueueLimit), ctx: ctx, cancel: cancel, done: make(chan struct{}), result: result}
	go func() {
		defer close(q.done)
		for {
			select {
			case <-ctx.Done():
				q.mu.Lock()
				q.closed = true
				q.mu.Unlock()
				for {
					select {
					case message := <-q.queue:
						q.result(deliveryResult{message: message, err: context.Canceled})
					default:
						return
					}
				}
			case message := <-q.queue:
				q.result(deliver(ctx, writer, config, message))
			}
		}
	}()
	return q
}

func (q *deliveryQueue) Submit(topic string, payload []byte) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed || q.ctx.Err() != nil {
		return false
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return false
	}
	message := outputMessage{ID: fmt.Sprintf("%x", id), Topic: topic, Payload: append([]byte(nil), payload...)}
	select {
	case q.queue <- message:
		return true
	default:
		return false
	}
}

func (q *deliveryQueue) Close() {
	q.mu.Lock()
	q.closed = true
	q.cancel()
	q.mu.Unlock()
	<-q.done
	q.writer.Close()
}

func recordDelivery(result deliveryResult) {
	state.Lock()
	state.PendingCount--
	state.RetryCount += result.retries
	if result.uncertain {
		state.UncertainCount++
	}
	if result.err == nil {
		state.ForwardedCount++
		state.LastOutput = nowISO()
	} else {
		state.DroppedCount++
	}
	state.Unlock()
	if result.err != nil {
		addLog("WARN", "output", result.err.Error())
	} else {
		messageTraffic.Record(result.message.Topic, result.message.Payload)
	}
}

func publicOutputURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	u.User, u.RawQuery, u.Fragment = nil, "", ""
	return u.String()
}
