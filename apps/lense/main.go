package main

import (
	"log"
	"net/http"
)

func main() {
	if err := openDB(); err != nil {
		log.Fatal(err)
	}
	if err := ensureSchema(); err != nil {
		log.Fatal(err)
	}
	go startMQTT()
	startComponentMonitor()

	port := getenv("LENSE_PORT", "8000")

	// Static assets must be registered explicitly. Otherwise /static/standard-chart.js
	// is handled by the catch-all UI route and the browser receives HTML instead of JS.
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("/app/static"))))

	http.HandleFunc("/api/graph-traffic", apiGraphTraffic)
	http.HandleFunc("/api/live-messages", apiLiveMessages)
	http.HandleFunc("/api/summary", apiSummary)
	http.HandleFunc("/api/database/tables", apiDatabaseTables)
	http.HandleFunc("/api/database/query", apiDatabaseQuery)
	http.HandleFunc("/api/agents", apiAgents)
	http.HandleFunc("/api/agents/", apiAgent)
	http.HandleFunc("/api/catalog", apiCatalog)
	http.HandleFunc("/api/query", apiQuery)
	http.HandleFunc("/api/metric-values", apiMetricValues)
	http.HandleFunc("/api/availability", apiAvailability)
	http.HandleFunc("/api/uns", apiUNS)
	http.HandleFunc("/api/export", apiExport)
	http.HandleFunc("/api/runtime", apiRuntime)
	http.HandleFunc("/api/topology", apiPublicTopology)
	http.HandleFunc("/api/remote-config/apply", apiRemoteApplyConfig)
	http.HandleFunc("/api/remote-config/validate", apiRemoteValidateConfig)
	http.HandleFunc("/api/remote-config/history", apiRemoteConfigHistory)
	http.HandleFunc("/api/remote-config", apiRemoteConfig)
	http.HandleFunc("/api/logs", apiLogs)
	http.Handle("/protocol-lab/", protocolLabProxy(getenv("PROTOCOL_LAB_URL", "http://protocol-lab:8500")))
	http.HandleFunc("/protocols", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/protocol-lab/static/?"+r.URL.RawQuery, http.StatusTemporaryRedirect)
	})
	http.HandleFunc("/", ui)

	log.Println("IoT Lense listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
