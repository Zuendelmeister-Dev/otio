package main

type MetricMessage struct {
	SchemaVersion string `json:"schemaVersion"`
	Timestamp     string `json:"timestamp"`
	AgentID       string `json:"agentId"`
	Source        struct {
		Type    string `json:"type"`
		Host    string `json:"host"`
		Address string `json:"address"`
	} `json:"source"`
	Metric struct {
		Name  string `json:"name"`
		Value any    `json:"value"`
		Unit  string `json:"unit"`
		Type  string `json:"type"`
	} `json:"metric"`
	Quality struct {
		Status string `json:"status"`
	} `json:"quality"`
}

type StatusMessage struct {
	Timestamp string `json:"timestamp"`
	AgentID   string `json:"agentId"`
	Connected bool   `json:"connected"`
	Healthy   bool   `json:"healthy"`
	Source    struct {
		Type string `json:"type"`
		Host string `json:"host"`
	} `json:"source"`
}

type ErrorMessage struct {
	Timestamp string `json:"timestamp"`
	AgentID   string `json:"agentId"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
}
