package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func requireConfigWriteToken(w http.ResponseWriter, r *http.Request) bool {
	if os.Getenv("OTIO_CONFIG_READ_ONLY") == "true" {
		http.Error(w, "HA configuration is managed by Helm. Update the shared configuration and roll out all replicas.", http.StatusForbidden)
		return false
	}
	token := strings.TrimSpace(os.Getenv("OTIO_CONFIG_WRITE_TOKEN"))
	if token == "" || r.Header.Get("X-OTIO-Config-Token") == token {
		return true
	}
	http.Error(w, "configuration write token required", http.StatusUnauthorized)
	return false
}

func apiStatus(w http.ResponseWriter, r *http.Request) {
	state.Lock()
	broker := map[string]any{"connected": state.BrokerConnected, "lastPublish": state.LastPublish, "publishCount": state.PublishCount}
	config := map[string]any{"loaded": state.ConfigLoaded, "valid": state.ConfigValid, "path": state.ConfigPath, "error": state.ConfigError, "clientId": state.Config.Broker.ClientID, "topicPrefix": state.Config.Broker.TopicPrefix, "brokerHost": state.Config.Broker.Host, "brokerPort": state.Config.Broker.Port, "revision": configRevision(state.ConfigRaw), "update": configUpdateLocked()}
	state.Unlock()
	writeJSON(w, map[string]any{"broker": broker, "config": config, "sources": snapshotSources()})
}

func apiMetrics(w http.ResponseWriter, r *http.Request) {
	state.Lock()
	defer state.Unlock()
	writeJSON(w, state.Metrics)
}

func apiLogs(w http.ResponseWriter, r *http.Request) {
	state.Lock()
	defer state.Unlock()
	writeJSON(w, map[string]any{"items": state.Logs})
}

func apiConfig(w http.ResponseWriter, r *http.Request) {
	state.Lock()
	defer state.Unlock()
	writeJSON(w, map[string]any{"raw": state.ConfigRaw, "parsed": state.Config, "revision": configRevision(state.ConfigRaw), "update": configUpdateLocked()})
}

func apiHistory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"items": listHistory()})
}

func apiValidateConfig(w http.ResponseWriter, r *http.Request) {
	if !requireConfigWriteToken(w, r) {
		return
	}
	var request struct {
		Raw string `json:"raw"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	normalized, err := normalizeJSON(request.Raw)
	if err != nil {
		writeJSON(w, map[string]any{"valid": false, "problems": []string{err.Error()}})
		return
	}
	_, problems := parseAndValidate(normalized)
	if len(problems) > 0 {
		writeJSON(w, map[string]any{"valid": false, "problems": problems})
		return
	}
	state.Lock()
	oldRaw := state.ConfigRaw
	state.Unlock()
	proposal := Proposal{ID: randomID(), CreatedAt: nowISO(), ConfigRaw: normalized, Diff: makeDiff(oldRaw, normalized), BaseRevision: configRevision(oldRaw)}
	state.Lock()
	state.Proposals[proposal.ID] = proposal
	state.Unlock()
	writeJSON(w, map[string]any{"valid": true, "problems": []string{}, "proposal": proposal})
}

func apiApplyConfig(w http.ResponseWriter, r *http.Request) {
	if !requireConfigWriteToken(w, r) {
		return
	}
	var request struct {
		ProposalID string `json:"proposalId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !lockConfigMutation(w) {
		return
	}
	defer configMutation.Unlock()
	state.Lock()
	proposal, ok := state.Proposals[request.ProposalID]
	state.Unlock()
	if !ok {
		http.Error(w, "proposal not found", http.StatusNotFound)
		return
	}
	historyFile, update, err := commitConfig(proposal.ConfigRaw, proposal.BaseRevision)
	if err != nil {
		writeConfigCommitError(w, err)
		return
	}
	state.Lock()
	delete(state.Proposals, request.ProposalID)
	state.Unlock()
	addLog("INFO", "config", "Applied config proposal "+request.ProposalID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]any{"status": "restarting", "saved": true, "historyFile": historyFile, "update": update})
}

func apiRollbackConfig(w http.ResponseWriter, r *http.Request) {
	if !requireConfigWriteToken(w, r) {
		return
	}
	var request struct {
		FileName     string `json:"fileName"`
		BaseRevision string `json:"baseRevision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.Contains(request.FileName, "..") || strings.ContainsAny(request.FileName, `/\\`) {
		http.Error(w, "invalid file name", http.StatusBadRequest)
		return
	}
	if !lockConfigMutation(w) {
		return
	}
	defer configMutation.Unlock()
	state.Lock()
	dir := state.HistoryDir
	revision := configRevision(state.ConfigRaw)
	state.Unlock()
	if request.BaseRevision != "" {
		revision = request.BaseRevision
	}
	raw, err := os.ReadFile(filepath.Join(dir, request.FileName))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	_, problems := parseAndValidate(string(raw))
	if len(problems) > 0 {
		http.Error(w, strings.Join(problems, "; "), http.StatusBadRequest)
		return
	}
	normalized, err := normalizeJSON(string(raw))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	historyFile, update, err := commitConfig(normalized, revision)
	if err != nil {
		writeConfigCommitError(w, err)
		return
	}
	addLog("WARN", "config", "Rolled back to "+request.FileName)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]any{"status": "restarting", "saved": true, "historyFile": historyFile, "update": update})
}

func lockConfigMutation(w http.ResponseWriter) bool {
	if !configMutation.TryLock() {
		http.Error(w, "another configuration update is in progress", http.StatusConflict)
		return false
	}
	state.Lock()
	restarting := state.ConfigUpdate.Phase == "restarting"
	state.Unlock()
	if restarting {
		configMutation.Unlock()
		http.Error(w, "configuration is saved and workers are restarting; wait before applying another change", http.StatusConflict)
		return false
	}
	return true
}

func writeConfigCommitError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, errConfigConflict) {
		status = http.StatusConflict
	}
	http.Error(w, err.Error(), status)
}

func apiTestRead(w http.ResponseWriter, r *http.Request) {
	agentID := strings.TrimPrefix(r.URL.Path, "/api/sources/")
	agentID = strings.TrimSuffix(agentID, "/test-read")
	state.Lock()
	config := state.Config
	state.Unlock()
	for _, source := range config.Sources {
		if source.AgentID == agentID {
			writeJSON(w, testRead(source))
			return
		}
	}
	http.Error(w, "source not found", http.StatusNotFound)
}
