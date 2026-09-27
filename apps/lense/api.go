package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

func findRemoteBaseURL(id string) string {
	topology := loadRawTopology()
	groups := []string{"senses", "dispenses", "presenses", "agents"}
	for _, group := range groups {
		items, _ := topology[group].([]any)
		for _, rawItem := range items {
			item, _ := rawItem.(map[string]any)
			if item == nil {
				continue
			}
			itemID, _ := item["id"].(string)
			if itemID == "" {
				itemID, _ = item["agentId"].(string)
			}
			if itemID != id {
				continue
			}
			if internal, ok := item["internalUrl"].(string); ok && internal != "" {
				return strings.TrimRight(internal, "/")
			}
			if external, ok := item["url"].(string); ok && external != "" {
				return strings.TrimRight(external, "/")
			}
		}
	}
	return ""
}

func proxyJSON(w http.ResponseWriter, r *http.Request, method string, url string, body []byte) {
	request, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	if token := r.Header.Get("X-OTIO-Config-Token"); token != "" {
		request.Header.Set("X-OTIO-Config-Token", token)
	}
	client := http.Client{Timeout: 3 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	payload, _ := io.ReadAll(response.Body)
	w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	w.WriteHeader(response.StatusCode)
	_, _ = w.Write(payload)
}

func apiRemoteConfig(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	base := findRemoteBaseURL(id)
	if base == "" {
		http.Error(w, "remote component not found", http.StatusNotFound)
		return
	}
	proxyJSON(w, r, http.MethodGet, base+"/api/config", nil)
}

func apiRemoteConfigHistory(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	base := findRemoteBaseURL(id)
	if base == "" {
		http.Error(w, "remote component not found", http.StatusNotFound)
		return
	}
	proxyJSON(w, r, http.MethodGet, base+"/api/config/history", nil)
}

func apiRemoteValidateConfig(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	base := findRemoteBaseURL(id)
	if base == "" {
		http.Error(w, "remote component not found", http.StatusNotFound)
		return
	}
	body, _ := io.ReadAll(r.Body)
	proxyJSON(w, r, http.MethodPost, base+"/api/config/validate", body)
}

func apiRemoteApplyConfig(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	base := findRemoteBaseURL(id)
	if base == "" {
		http.Error(w, "remote component not found", http.StatusNotFound)
		return
	}
	body, _ := io.ReadAll(r.Body)
	proxyJSON(w, r, http.MethodPost, base+"/api/config/apply", body)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func scanTimeString(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func nullStringValue(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func calcStats(values []float64) (float64, float64, float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}
	min := values[0]
	max := values[0]
	sum := 0.0
	for _, value := range values {
		sum += value
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
	}
	return sum / float64(len(values)), min, max
}

func normalizeAggregation(input string) string {
	switch strings.ToLower(input) {
	case "avg", "min", "max", "sum":
		return strings.ToLower(input)
	default:
		return "avg"
	}
}

func messageFromPayload(raw string, fallback string) string {
	if raw == "" {
		return fallback
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return fallback
	}
	if message, ok := payload["message"].(string); ok && message != "" {
		return message
	}
	return fallback
}

func activeIssueItems(componentStatuses map[string]ComponentStatus) []map[string]any {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var items []map[string]any

	ids := configuredAgentIDs()
	if len(ids) > 0 {
		rows, _ := db.Query(
			`SELECT agent_id, ts, connected, healthy, payload FROM agent_status WHERE agent_id IN (`+sqlInPlaceholders(len(ids), 1)+`) AND (connected = false OR healthy = false) ORDER BY ts DESC`,
			stringArgs(ids)...,
		)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var agentID string
				var ts time.Time
				var connected, healthy bool
				var payload string
				_ = rows.Scan(&agentID, &ts, &connected, &healthy, &payload)
				message := messageFromPayload(payload, "Agent is not healthy")
				if !connected {
					message = "Agent is disconnected: " + message
				} else if !healthy {
					message = "Agent is unhealthy: " + message
				}
				items = append(items, map[string]any{
					"timestamp": ts.UTC().Format(time.RFC3339Nano),
					"firstSeen": now,
					"lastSeen":  ts.UTC().Format(time.RFC3339Nano),
					"level":     "ERROR",
					"component": agentID,
					"message":   message,
					"count":     1,
					"active":    true,
				})
			}
		}
	}

	for _, status := range componentStatuses {
		if status.Connected && status.Healthy {
			continue
		}
		message := status.Message
		if message == "" {
			message = "Component is not healthy"
		}
		items = append(items, map[string]any{
			"timestamp": status.LastSeen,
			"firstSeen": status.LastSeen,
			"lastSeen":  status.LastSeen,
			"level":     "ERROR",
			"component": status.ID,
			"message":   message,
			"count":     1,
			"active":    true,
		})
	}
	return items
}

func apiSummary(w http.ResponseWriter, r *http.Request) {
	configured := configuredAgentIDs()
	componentStatuses := componentStatusSnapshot()
	agentCount := len(configured)
	connected, healthy, unhealthy := 0, 0, 0

	if agentCount > 0 {
		query := `SELECT agent_id, connected, healthy FROM agent_status WHERE agent_id IN (` + sqlInPlaceholders(agentCount, 1) + `)`
		rows, _ := db.Query(query, stringArgs(configured)...)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var agentID string
				var c, h bool
				_ = rows.Scan(&agentID, &c, &h)
				if topo, ok := configuredAgentSet()[agentID]; ok && !isComponentHealthy(componentStatuses, topo.SenseID, true) {
					c = false
					h = false
				}
				if c {
					connected++
				}
				if h {
					healthy++
				} else {
					unhealthy++
				}
			}
		}
	}

	var counts []float64
	if agentCount > 0 {
		countParams := stringArgs(configured)
		rows, _ := db.Query(`SELECT agent_id, COUNT(*) FROM metric_events WHERE ts > NOW() - INTERVAL '5 minutes' AND agent_id IN (`+sqlInPlaceholders(agentCount, 1)+`) GROUP BY agent_id`, countParams...)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var agent string
				var count float64
				_ = rows.Scan(&agent, &count)
				counts = append(counts, count)
			}
		}
	}
	avg, min, max := calcStats(counts)

	errors := activeIssueItems(componentStatuses)

	state.Lock()
	mqtt := map[string]any{"connected": state.BrokerConnected, "lastMessage": state.LastMessage, "messageCount": state.MessageCount}
	state.Unlock()
	writeJSON(w, map[string]any{"mqtt": mqtt, "connectedAgents": connected, "health": map[string]any{"healthy": healthy, "unhealthy": unhealthy}, "messagesPerAgent5m": map[string]any{"avg": avg, "min": min, "max": max}, "latestErrors": errors, "componentStatus": componentStatuses})
}

func apiAgents(w http.ResponseWriter, r *http.Request) {
	search := strings.ToLower(r.URL.Query().Get("search"))
	configured := configuredAgentSet()
	componentStatuses := componentStatusSnapshot()
	ids := configuredAgentIDs()
	statusByID := map[string]map[string]any{}

	if len(ids) > 0 {
		query := `SELECT agent_id, ts, connected, healthy, source_type, source_host, payload FROM agent_status WHERE agent_id IN (` + sqlInPlaceholders(len(ids), 1) + `)`
		rows, _ := db.Query(query, stringArgs(ids)...)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var agent string
				var ts time.Time
				var sourceType, sourceHost sql.NullString
				var payload string
				var connected, healthy bool
				_ = rows.Scan(&agent, &ts, &connected, &healthy, &sourceType, &sourceHost, &payload)
				statusByID[agent] = map[string]any{"agentId": agent, "timestamp": scanTimeString(ts), "connected": connected, "healthy": healthy, "sourceType": nullStringValue(sourceType), "sourceHost": nullStringValue(sourceHost), "payload": json.RawMessage(payload)}
			}
		}
	}

	var items []map[string]any
	for _, topo := range loadTopology().Agents {
		if search != "" && !strings.Contains(strings.ToLower(topo.AgentID), search) && !strings.Contains(strings.ToLower(topo.SourceHost), search) && !strings.Contains(strings.ToLower(topo.SourceType), search) {
			continue
		}
		item := map[string]any{
			"agentId":    topo.AgentID,
			"timestamp":  "",
			"connected":  false,
			"healthy":    false,
			"sourceType": topo.SourceType,
			"sourceHost": topo.SourceHost,
			"senseId":    topo.SenseID,
			"payload":    json.RawMessage(`{}`),
		}
		if status, ok := statusByID[topo.AgentID]; ok {
			for key, value := range status {
				item[key] = value
			}
			item["senseId"] = configured[topo.AgentID].SenseID
		}
		if topo.SenseID != "" && !isComponentHealthy(componentStatuses, topo.SenseID, true) {
			item["connected"] = false
			item["healthy"] = false
			item["payload"] = json.RawMessage(`{"message":"Owning IoT Sense instance is offline"}`)
		}
		items = append(items, item)
	}
	writeJSON(w, map[string]any{"items": items})
}

func apiAgent(w http.ResponseWriter, r *http.Request) {
	agentID := strings.TrimPrefix(r.URL.Path, "/api/agents/")
	topo, configured := configuredAgentSet()[agentID]
	sourceIDs := []string{agentID}
	if !configured {
		if sense, exists := configuredSenseSet()[agentID]; exists {
			sourceIDs = []string{}
			for _, source := range loadTopology().Agents {
				if source.SenseID == agentID {
					sourceIDs = append(sourceIDs, source.AgentID)
				}
			}
			topo = AgentTopology{AgentID: agentID, SourceType: "sense", SourceHost: sense.URL}
			configured = true
		}
	}
	if !configured {
		http.Error(w, "agent is not in current topology", http.StatusNotFound)
		return
	}

	row := db.QueryRow(`SELECT agent_id, ts, connected, healthy, source_type, source_host, payload FROM agent_status WHERE agent_id=$1`, agentID)
	var agent string
	var ts time.Time
	var sourceType, sourceHost sql.NullString
	var payload string
	var connected, healthy bool
	err := row.Scan(&agent, &ts, &connected, &healthy, &sourceType, &sourceHost, &payload)
	if err != nil {
		agent = topo.AgentID
		sourceType = sql.NullString{String: topo.SourceType, Valid: true}
		sourceHost = sql.NullString{String: topo.SourceHost, Valid: true}
		payload = `{}`
	}

	topicRows, _ := db.Query(`SELECT DISTINCT topic FROM metric_events WHERE agent_id=ANY($1) ORDER BY topic`, pq.Array(sourceIDs))
	var topics []string
	if topicRows != nil {
		defer topicRows.Close()
		for topicRows.Next() {
			var topic string
			_ = topicRows.Scan(&topic)
			topics = append(topics, topic)
		}
	}

	msgRows, _ := db.Query(`SELECT ts, topic, payload FROM metric_events WHERE agent_id=ANY($1) ORDER BY ts DESC LIMIT 50`, pq.Array(sourceIDs))
	var messages []map[string]any
	if msgRows != nil {
		defer msgRows.Close()
		for msgRows.Next() {
			var msgTS time.Time
			var topic string
			var msgPayload string
			_ = msgRows.Scan(&msgTS, &topic, &msgPayload)
			messages = append(messages, map[string]any{"timestamp": scanTimeString(msgTS), "topic": topic, "payload": json.RawMessage(msgPayload)})
		}
	}

	if topo.SourceType == "sense" {
		state := componentStatusSnapshot()[agentID]
		connected = state.Connected
		healthy = state.Healthy
	}
	timestamp := ""
	if !ts.IsZero() {
		timestamp = scanTimeString(ts)
	}
	writeJSON(w, map[string]any{
		"agent":    map[string]any{"agentId": agent, "timestamp": timestamp, "connected": connected, "healthy": healthy, "sourceType": nullStringValue(sourceType), "sourceHost": nullStringValue(sourceHost), "senseId": topo.SenseID, "payload": json.RawMessage(payload)},
		"topics":   topics,
		"messages": messages,
	})
}

func apiCatalog(w http.ResponseWriter, r *http.Request) {
	var agents, metrics []string
	for _, agent := range loadTopology().Agents {
		agents = append(agents, agent.AgentID)
	}

	ids := configuredAgentIDs()
	if len(ids) > 0 {
		rows, _ := db.Query(`SELECT DISTINCT metric_name FROM metric_events WHERE agent_id IN (`+sqlInPlaceholders(len(ids), 1)+`) ORDER BY metric_name`, stringArgs(ids)...)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var value string
				_ = rows.Scan(&value)
				metrics = append(metrics, value)
			}
		}
	}
	writeJSON(w, map[string]any{"agents": agents, "metrics": metrics})
}

func apiQuery(w http.ResponseWriter, r *http.Request) {
	agent := r.URL.Query().Get("agent_id")
	metric := r.URL.Query().Get("metric_name")
	agg := normalizeAggregation(r.URL.Query().Get("aggregation"))
	rangeMinutes, _ := strconv.Atoi(r.URL.Query().Get("range_minutes"))
	if rangeMinutes <= 0 {
		rangeMinutes = 60
	}
	bucket, _ := strconv.Atoi(r.URL.Query().Get("bucket_seconds"))
	if bucket <= 0 {
		bucket = 60
	}
	ids := configuredAgentIDs()
	params := []any{bucket, bucket, rangeMinutes}
	where := `ts >= NOW() - ($3 * INTERVAL '1 minute') AND metric_value IS NOT NULL`
	index := 4
	if agent == "" || agent == "all" {
		if len(ids) > 0 {
			where += ` AND agent_id IN (` + sqlInPlaceholders(len(ids), index) + `)`
			for _, id := range ids {
				params = append(params, id)
				index++
			}
		}
	} else if agent != "" && agent != "all" {
		where += ` AND agent_id = $` + strconv.Itoa(index)
		params = append(params, agent)
		index++
	}
	if metric != "" && metric != "all" {
		where += ` AND metric_name = $` + strconv.Itoa(index)
		params = append(params, metric)
	}
	query := `SELECT to_timestamp(floor(extract(epoch from ts) / $1) * $2) AS bucket_ts, agent_id, metric_name, ` + agg + `(metric_value) AS value FROM metric_events WHERE ` + where + ` GROUP BY bucket_ts, agent_id, metric_name ORDER BY bucket_ts ASC, agent_id ASC, metric_name ASC`
	rows, _ := db.Query(query, params...)
	var items []map[string]any
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var ts time.Time
			var agentID, metricName string
			var value float64
			_ = rows.Scan(&ts, &agentID, &metricName, &value)
			items = append(items, map[string]any{"timestamp": scanTimeString(ts), "agentId": agentID, "metricName": metricName, "value": value})
		}
	}
	writeJSON(w, map[string]any{"items": items})
}

func apiMetricValues(w http.ResponseWriter, r *http.Request) {
	agent := r.URL.Query().Get("agent_id")
	metric := r.URL.Query().Get("metric_name")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if agent == "" || metric == "" {
		writeJSON(w, map[string]any{"items": []map[string]any{}})
		return
	}
	rows, _ := db.Query(
		`SELECT ts, metric_value, metric_text, metric_unit, metric_type
		 FROM metric_events
		 WHERE agent_id=$1 AND metric_name=$2
		 ORDER BY ts DESC
		 LIMIT $3`,
		agent,
		metric,
		limit,
	)
	var items []map[string]any
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var ts time.Time
			var numeric sql.NullFloat64
			var textValue, unit, metricType sql.NullString
			_ = rows.Scan(&ts, &numeric, &textValue, &unit, &metricType)
			var value any
			valueKind := "text"
			if numeric.Valid {
				value = numeric.Float64
				valueKind = "number"
			} else {
				value = nullStringValue(textValue)
			}
			items = append(items, map[string]any{
				"timestamp":  scanTimeString(ts),
				"agentId":    agent,
				"metricName": metric,
				"value":      value,
				"valueKind":  valueKind,
				"unit":       nullStringValue(unit),
				"metricType": nullStringValue(metricType),
			})
		}
	}
	writeJSON(w, map[string]any{"items": items})
}

func apiUNS(w http.ResponseWriter, r *http.Request) {
	search := strings.ToLower(r.URL.Query().Get("search"))
	configured := configuredAgentSet()
	state.Lock()
	var topics []string
	for topic := range state.Topics {
		parts := strings.Split(topic, "/")
		if len(parts) >= 2 {
			if _, ok := configured[parts[1]]; !ok && parts[1] != "_sense" {
				continue
			}
		}
		if search == "" || strings.Contains(strings.ToLower(topic), search) {
			topics = append(topics, topic)
		}
	}
	state.Unlock()

	prefix := getenv("MQTT_TOPIC_PREFIX", "iot-lense")
	for _, agent := range loadTopology().Agents {
		fallback := []string{
			prefix + "/" + agent.AgentID + "/status",
			prefix + "/" + agent.AgentID + "/errors",
		}
		for _, topic := range fallback {
			if search == "" || strings.Contains(strings.ToLower(topic), search) {
				topics = append(topics, topic)
			}
		}
	}
	writeJSON(w, map[string]any{"topics": topics, "tree": buildTree(topics)})
}

func buildTree(topics []string) map[string]any {
	root := map[string]any{}
	for _, topic := range topics {
		node := root
		parts := strings.Split(topic, "/")
		for _, part := range parts {
			if _, ok := node[part]; !ok {
				node[part] = map[string]any{}
			}
			node = node[part].(map[string]any)
		}
	}
	return root
}

func apiExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	rows, _ := db.Query(`SELECT ts, agent_id, metric_name, metric_value, metric_text, metric_unit FROM metric_events ORDER BY ts DESC LIMIT 1000`)
	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("timestamp,agent_id,metric_name,metric_value,metric_text,metric_unit\n"))
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var ts time.Time
				var agent, metric string
				var unit, textValue sql.NullString
				var numericValue sql.NullFloat64
				_ = rows.Scan(&ts, &agent, &metric, &numericValue, &textValue, &unit)
				valueText := ""
				if numericValue.Valid {
					valueText = strconv.FormatFloat(numericValue.Float64, 'f', -1, 64)
				}
				_, _ = w.Write([]byte(scanTimeString(ts) + "," + agent + "," + metric + "," + valueText + "," + nullStringValue(textValue) + "," + nullStringValue(unit) + "\n"))
			}
		}
		return
	}
	var items []map[string]any
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var ts time.Time
			var agent, metric string
			var unit, textValue sql.NullString
			var numericValue sql.NullFloat64
			_ = rows.Scan(&ts, &agent, &metric, &numericValue, &textValue, &unit)
			var value any
			if numericValue.Valid {
				value = numericValue.Float64
			} else {
				value = nullStringValue(textValue)
			}
			items = append(items, map[string]any{"timestamp": scanTimeString(ts), "agentId": agent, "metricName": metric, "value": value, "textValue": nullStringValue(textValue), "unit": nullStringValue(unit)})
		}
	}
	writeJSON(w, map[string]any{"items": items})
}

func apiAvailability(w http.ResponseWriter, r *http.Request) {
	agent := r.URL.Query().Get("agent_id")
	rangeMinutes, _ := strconv.Atoi(r.URL.Query().Get("range_minutes"))
	if rangeMinutes <= 0 {
		rangeMinutes = 60
	}
	bucketSeconds, _ := strconv.Atoi(r.URL.Query().Get("bucket_seconds"))
	if bucketSeconds <= 0 {
		bucketSeconds = 60
	}

	if agent == "" || agent == "all" || !isConfiguredAgent(agent) {
		writeJSON(w, map[string]any{"items": []map[string]any{}})
		return
	}

	rows, _ := db.Query(
		`SELECT to_timestamp(floor(extract(epoch from ts) / $1) * $2) AS bucket_ts, COUNT(*) AS count
		 FROM metric_events
		 WHERE agent_id=$3 AND ts >= NOW() - ($4 * INTERVAL '1 minute')
		 GROUP BY bucket_ts
		 ORDER BY bucket_ts ASC`,
		bucketSeconds,
		bucketSeconds,
		agent,
		rangeMinutes,
	)

	var items []map[string]any
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var ts time.Time
			var count int
			_ = rows.Scan(&ts, &count)
			items = append(items, map[string]any{
				"timestamp": scanTimeString(ts),
				"agentId":   agent,
				"up":        count > 0,
				"value":     count,
			})
		}
	}
	writeJSON(w, map[string]any{"items": items})
}

func apiRuntime(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"mqtt": map[string]any{
			"host":        getenv("MQTT_BROKER", "mqtt"),
			"port":        getenv("MQTT_PORT", "1883"),
			"topicPrefix": getenv("MQTT_TOPIC_PREFIX", "iot-lense"),
		},
		"postgres": map[string]any{
			"host":     getenv("POSTGRES_HOST", "postgres"),
			"port":     getenv("POSTGRES_PORT", "5432"),
			"database": getenv("POSTGRES_DB", "iotdb"),
			"user":     getenv("POSTGRES_USER", "iot"),
		},
		"topology":        loadRawTopology(),
		"componentStatus": componentStatusSnapshot(),
	})
}

func apiLogs(w http.ResponseWriter, r *http.Request) {
	items := activeIssueItems(componentStatusSnapshot())
	rows, _ := db.Query(`
		SELECT
			COALESCE(agent_id,'') AS component,
			UPPER(COALESCE(severity,'error')) AS level,
			message,
			MIN(ts) AS first_seen,
			MAX(ts) AS last_seen,
			COUNT(*) AS count
		FROM error_events
		WHERE ts >= NOW() - INTERVAL '24 hours'
		GROUP BY COALESCE(agent_id,''), UPPER(COALESCE(severity,'error')), message
		ORDER BY MAX(ts) DESC
		LIMIT 500
	`)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var component, level, message string
			var firstSeen, lastSeen time.Time
			var count int
			_ = rows.Scan(&component, &level, &message, &firstSeen, &lastSeen, &count)
			items = append(items, map[string]any{
				"timestamp": lastSeen.UTC().Format(time.RFC3339Nano),
				"firstSeen": firstSeen.UTC().Format(time.RFC3339Nano),
				"lastSeen":  lastSeen.UTC().Format(time.RFC3339Nano),
				"level":     level,
				"component": component,
				"message":   message,
				"count":     count,
			})
		}
	}
	state.Lock()
	if state.BrokerConnected {
		items = append([]map[string]any{{
			"timestamp": state.LastMessage,
			"firstSeen": state.LastMessage,
			"lastSeen":  state.LastMessage,
			"level":     "INFO",
			"component": "mqtt",
			"message":   "Broker connection is up",
			"count":     1,
		}}, items...)
	} else {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		items = append([]map[string]any{{
			"timestamp": now,
			"firstSeen": now,
			"lastSeen":  now,
			"level":     "ERROR",
			"component": "mqtt",
			"message":   "Broker connection is down",
			"count":     1,
		}}, items...)
	}
	state.Unlock()
	writeJSON(w, map[string]any{"items": items})
}
