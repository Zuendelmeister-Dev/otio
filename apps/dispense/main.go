package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func ui(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "/app/static/index.html")
}

func main() {
	config := loadConfig()
	if err := validateOutputConfig(config); err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	state.Lock()
	state.Config = config
	state.Unlock()

	addLog("INFO", "startup", "Starting "+config.InstanceID+" for "+config.SourceFilter)
	var writer outputWriter
	if config.Output.Type == "http" {
		output, err := newHTTPOutput(config.Output)
		if err != nil {
			log.Fatal(err)
		}
		writer = output
	} else {
		output := connectOutput(config)
		state.Lock()
		state.OutputClient = output
		state.Unlock()
		writer = &mqttOutput{client: output}
	}
	queue := newDeliveryQueue(ctx, writer, config, recordDelivery)
	defer queue.Close()
	state.Lock()
	state.Output = queue
	state.Unlock()
	// Install the output before subscribing; retained input must not hit an
	// uninitialized output while the target broker is still being connected.
	input := connectInput(config)
	defer input.Disconnect(0)

	state.Lock()
	state.InputClient = input
	state.Unlock()

	port := getenv("DISPENSE_UI_PORT", "8200")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("/app/static"))))
	http.Handle("/api/traffic", &messageTraffic)
	http.HandleFunc("/api/messages", messageTraffic.ServeMessages)
	http.HandleFunc("/api/status", apiStatus)
	http.HandleFunc("/api/metrics", apiMetrics)
	http.HandleFunc("/api/logs", apiLogs)
	http.HandleFunc("/api/runtime", apiRuntime)
	http.HandleFunc("/", ui)

	addLog("INFO", "ui", "IoT Dispense UI listening on :"+port)
	server := &http.Server{Addr: ":" + port, Handler: nil, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Print(err)
	}
}
