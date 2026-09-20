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
	mux := http.NewServeMux()
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
		writeJSON(w, 200, map[string]any{"values": sim.snapshot(), "publishers": sim.publisherStatus(), "host": host, "mqttURL": os.Getenv("LAB_MQTT_URL"), "amqpHost": amqpHost, "amqpEnabled": os.Getenv("LAB_AMQP_URL") != ""})
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
			defer func() { <-slots }()
		default:
			writeJSON(w, 429, map[string]any{"error": "All eight test slots are busy; retry shortly."})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), readTimeout)
		defer cancel()
		start := time.Now()
		value, err := read(ctx, request)
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
	mux.Handle("GET /static/", http.FileServerFS(assets))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/", http.StatusTemporaryRedirect)
	})
	return mux
}
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	sim := newSimulator()
	uaServer, err := startOPCUA(ctx, sim, "0.0.0.0", 4842)
	if err != nil {
		log.Fatal(err)
	}
	defer uaServer.Close()
	tcp, err := listenModbus(ctx, sim, 1502, false)
	if err != nil {
		log.Fatal(err)
	}
	defer tcp.Close()
	rtu, err := listenModbus(ctx, sim, 1503, true)
	if err != nil {
		log.Fatal(err)
	}
	defer rtu.Close()
	go publishDemo(ctx, sim)
	srv := &http.Server{Addr: ":8500", Handler: handler(sim, readValue), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	log.Println("OT.io Protocol Lab: http://localhost:8500/static/ · Modbus :1502 · RTU tunnel :1503 · OPC UA :4842")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
