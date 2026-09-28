package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

//go:embed static/*
var assets embed.FS

type readFunc func(context.Context, ReadRequest) (any, error)

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("expected exactly one JSON object")
	}
	return nil
}
func handler(sim *simulator, read readFunc) http.Handler {
	return handlerWithRuntime(sim, read, newSimulationRuntime(context.Background(), sim))
}
func handlerWithRuntime(sim *simulator, read readFunc, runtime *simulationRuntime) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/certificates", certificatesAPI)
	mux.HandleFunc("PUT /api/simulator/listener", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Protocol string `json:"protocol"`
			Enabled  bool   `json:"enabled"`
		}
		if err := decode(w, r, &request); err != nil {
			writeJSON(w, 400, map[string]any{"error": err.Error()})
			return
		}
		if err := runtime.set(request.Protocol, request.Enabled); err != nil {
			writeJSON(w, 422, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"active": runtime.names()})
	})
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"status": "ok", "values": sim.snapshot(), "publishers": sim.publisherStatus()})
	})
	mux.HandleFunc("GET /api/catalog", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"items": catalog}) })
	mux.HandleFunc("GET /api/simulator", func(w http.ResponseWriter, r *http.Request) {
		host := os.Getenv("LAB_HOST")
		if host == "" {
			host = "protocol-lab"
		}
		amqpHost := "rabbitmq:5672"
		if endpoint, err := url.Parse(os.Getenv("LAB_AMQP_URL")); err == nil && endpoint.Host != "" {
			amqpHost = endpoint.Host
		}
		writeJSON(w, 200, map[string]any{"active": runtime.names(), "values": sim.snapshot(), "publishers": sim.publisherStatus(), "host": host, "mqttURL": os.Getenv("LAB_MQTT_URL"), "amqpHost": amqpHost, "amqpEnabled": os.Getenv("LAB_AMQP_URL") != ""})
	})
	mux.HandleFunc("PUT /api/simulator", func(w http.ResponseWriter, r *http.Request) {
		var v simValues
		if err := decode(w, r, &v); err != nil {
			writeJSON(w, 400, map[string]any{"error": err.Error()})
			return
		}
		if err := sim.update(v); err != nil {
			writeJSON(w, 400, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, 200, sim.snapshot())
	})
	slots := make(chan struct{}, 8)
	mux.HandleFunc("POST /api/read", func(w http.ResponseWriter, r *http.Request) {
		var request ReadRequest
		if err := decode(w, r, &request); err != nil {
			writeJSON(w, 400, map[string]any{"error": err.Error()})
			return
		}
		if err := validateRead(request); err != nil {
			writeJSON(w, 422, map[string]any{"error": err.Error()})
			return
		}
		select {
		case slots <- struct{}{}:
		default:
			writeJSON(w, 429, map[string]any{"error": "All eight test slots are busy; retry shortly."})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), readTimeout)
		defer cancel()
		start := time.Now()
		// Some experimental drivers do not finish their handshake on cancellation.
		// Bound the HTTP wait while retaining the slot until the worker exits.
		type result struct {
			value any
			err   error
		}
		done := make(chan result, 1)
		go func() { defer func() { <-slots }(); v, err := read(ctx, request); done <- result{v, err} }()
		var value any
		var err error
		select {
		case response := <-done:
			value, err = response.value, response.err
		case <-ctx.Done():
			err = ctx.Err()
		}
		if err != nil {
			status := 502
			if errors.Is(err, context.DeadlineExceeded) {
				status = 504
			}
			writeJSON(w, status, map[string]any{"error": err.Error(), "protocol": request.Protocol})
			return
		}
		writeJSON(w, 200, map[string]any{"value": value, "protocol": request.Protocol, "elapsedMs": time.Since(start).Milliseconds(), "timestamp": time.Now().UTC()})
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"status": "ok"}) })
	mux.Handle("GET /static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		http.FileServerFS(assets).ServeHTTP(w, r)
	}))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/", http.StatusTemporaryRedirect)
	})
	return mux
}
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	sim := newSimulator()
	runtime := newSimulationRuntime(ctx, sim)
	defer runtime.stopAll()
	for _, name := range strings.Split(os.Getenv("LAB_SIMULATORS"), ",") {
		if name = strings.TrimSpace(name); name != "" {
			if err := runtime.set(name, true); err != nil {
				log.Fatal(err)
			}
		}
	}
	httpAddress := os.Getenv("LAB_HTTP_ADDRESS")
	if httpAddress == "" {
		httpAddress = ":8500"
	}
	srv := &http.Server{Addr: httpAddress, Handler: handlerWithRuntime(sim, readValue, runtime), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	log.Printf("OT.io Protocol Lab listening on %s · diagnostic reads; simulators are opt-in", httpAddress)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
