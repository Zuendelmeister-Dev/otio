package main

import (
	"log"
	"net/http"
)

func ui(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "/app/static/index.html")
}

func main() {
	config := loadConfig()
	state.Lock()
	state.Config = config
	state.Unlock()

	addLog("INFO", "startup", "Starting "+config.InstanceID+" for "+config.SourceFilter)
	input := connectInput(config)
	output := connectOutput(config)

	state.Lock()
	state.InputClient = input
	state.OutputClient = output
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
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
