package app

import (
	"log"
	"net/http"
	"os"
)

func Run() {
	state.ConfigPath = getenv("SENSE_CONFIG_PATH", "/app/config/config.json")
	state.HistoryDir = getenv("SENSE_HISTORY_DIR", "/app/config/history")
	_ = os.MkdirAll(state.HistoryDir, 0755)
	if loadConfigFromDisk() {
		restartWorkers()
	}

	port := getenv("SENSE_UI_PORT", "8100")

	// Static assets must be registered explicitly. Otherwise /static/standard-chart.js
	// is handled by the catch-all UI route and the browser receives HTML instead of JS.
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("/app/static"))))

	http.Handle("/api/traffic", &messageTraffic)
	http.HandleFunc("/api/messages", messageTraffic.ServeMessages)
	http.HandleFunc("/api/status", apiStatus)
	http.HandleFunc("/api/metrics", apiMetrics)
	http.HandleFunc("/api/logs", apiLogs)
	http.HandleFunc("/api/config", apiConfig)
	http.HandleFunc("/api/config/history", apiHistory)
	http.HandleFunc("/api/config/validate", apiValidateConfig)
	http.HandleFunc("/api/config/apply", apiApplyConfig)
	http.HandleFunc("/api/config/rollback", apiRollbackConfig)
	http.HandleFunc("/api/sources/", apiTestRead)
	http.HandleFunc("/", ui)

	addLog("INFO", "ui", "IoT Sense UI listening on :"+port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
