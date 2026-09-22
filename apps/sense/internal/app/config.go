package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"iot-lense-sense/sense/internal/protocols"
)

func getenv(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func normalizeJSON(raw string) (string, error) {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return "", err
	}
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func validateConfig(config Config) []string {
	var problems []string
	if config.Broker.Host == "" {
		problems = append(problems, "broker.host is required")
	}
	if config.Broker.Port <= 0 || config.Broker.Port > 65535 {
		problems = append(problems, "broker.port must be between 1 and 65535")
	}
	if config.Broker.ClientID == "" {
		problems = append(problems, "broker.clientId is required")
	}
	if config.Broker.TopicPrefix == "" {
		problems = append(problems, "broker.topicPrefix is required")
	}
	if config.PollIntervalMS < 100 {
		problems = append(problems, "pollIntervalMs must be at least 100")
	}
	if config.HealthTimeoutSeconds == 0 {
		config.HealthTimeoutSeconds = 300
	}
	if len(config.Sources) == 0 {
		problems = append(problems, "at least one source is required")
	}
	agents := map[string]bool{}
	for i, source := range config.Sources {
		prefix := fmt.Sprintf("sources[%d]", i)
		if source.AgentID == "" {
			problems = append(problems, prefix+".agentId is required")
		}
		if agents[source.AgentID] {
			problems = append(problems, prefix+".agentId must be unique")
		}
		agents[source.AgentID] = true
		if _, err := protocols.NewRegistry().Reader(source.Type); err != nil {
			problems = append(problems, prefix+".type is unsupported: "+source.Type)
		}
		if strings.HasPrefix(source.Type, "lab-") {
			connection, _ := source.Options["connection"].(string)
			if connection == "" {
				problems = append(problems, prefix+".options.connection is required for lab protocols")
			}
		}
		if source.Host == "" {
			problems = append(problems, prefix+".host is required")
		}
		if source.Port <= 0 || source.Port > 65535 {
			problems = append(problems, prefix+".port must be between 1 and 65535")
		}
		readMode := source.ReadMode
		if readMode == "" {
			readMode = "poll"
		}
		if readMode != "poll" && readMode != "subscription" {
			problems = append(problems, prefix+".readMode must be poll or subscription")
		}
		if source.Type != "opcua" && readMode == "subscription" {
			problems = append(problems, prefix+".readMode subscription is currently supported for opcua only")
		}
		if source.SubscriptionIntervalMS != 0 && source.SubscriptionIntervalMS < 250 {
			problems = append(problems, prefix+".subscriptionIntervalMs must be at least 250 when set")
		}
		if source.Type == "modbus-tcp" && source.UnitID == 0 {
			problems = append(problems, prefix+".unitId must be greater than 0 for modbus-tcp")
		}
		if len(source.Metrics) == 0 {
			problems = append(problems, prefix+".metrics must not be empty")
		}
		metrics := map[string]bool{}
		for j, metric := range source.Metrics {
			mprefix := fmt.Sprintf("%s.metrics[%d]", prefix, j)
			if metric.Name == "" {
				problems = append(problems, mprefix+".name is required")
			}
			if metrics[metric.Name] {
				problems = append(problems, mprefix+".name must be unique within a source")
			}
			metrics[metric.Name] = true
			if source.Type == "opcua" && metric.NodeID == "" {
				problems = append(problems, mprefix+".nodeId is required for opcua")
			}
			if strings.HasPrefix(source.Type, "lab-") && metric.Address == "" && metric.NodeID == "" {
				problems = append(problems, mprefix+".address or nodeId is required for lab protocols")
			}
			if metric.Scale == 0 {
				problems = append(problems, mprefix+".scale must not be 0")
			}
		}
	}
	return problems
}

func parseAndValidate(raw string) (Config, []string) {
	var config Config
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return config, []string{fmt.Sprintf("invalid JSON or unknown field: %v", err)}
	}
	return config, validateConfig(config)
}

func loadConfigFromDisk() bool {
	path := getenv("SENSE_CONFIG_PATH", "/app/config/config.json")
	historyDir := getenv("SENSE_HISTORY_DIR", "/app/config/history")
	rawBytes, err := os.ReadFile(path)
	if err != nil {
		state.Lock()
		state.ConfigPath = path
		state.HistoryDir = historyDir
		state.ConfigLoaded = false
		state.ConfigValid = false
		state.ConfigError = err.Error()
		state.Unlock()
		addLog("ERROR", "config", err.Error())
		return false
	}
	normalized, err := normalizeJSON(string(rawBytes))
	if err != nil {
		normalized = string(rawBytes)
	}
	config, problems := parseAndValidate(normalized)
	state.Lock()
	state.ConfigPath = path
	state.HistoryDir = historyDir
	state.ConfigRaw = normalized
	state.Config = config
	state.ConfigLoaded = true
	state.ConfigValid = len(problems) == 0
	state.ConfigError = strings.Join(problems, "; ")
	state.Unlock()
	if len(problems) > 0 {
		addLog("ERROR", "config", "Config validation failed: "+strings.Join(problems, "; "))
		return false
	}
	addLog("INFO", "config", "Loaded config from "+path)
	return true
}

func randomID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func makeDiff(oldText string, newText string) string {
	oldLines := strings.Split(oldText, "\n")
	newLines := strings.Split(newText, "\n")
	max := len(oldLines)
	if len(newLines) > max {
		max = len(newLines)
	}
	var b strings.Builder
	for i := 0; i < max; i++ {
		oldLine, newLine := "", ""
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}
		if oldLine == newLine {
			b.WriteString("  " + oldLine + "\n")
			continue
		}
		if i < len(oldLines) {
			b.WriteString("- " + oldLine + "\n")
		}
		if i < len(newLines) {
			b.WriteString("+ " + newLine + "\n")
		}
	}
	return b.String()
}

func saveHistorySnapshot(raw string) (string, error) {
	state.Lock()
	dir := state.HistoryDir
	state.Unlock()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	name := time.Now().UTC().Format("20060102T150405.000000000Z") + ".json"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(raw), 0644); err != nil {
		return "", err
	}
	return name, pruneHistory(dir, 5)
}

func pruneHistory(dir string, keep int) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var names []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			names = append(names, file.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	if keep < 0 {
		keep = 0
	}
	if len(names) <= keep {
		return nil
	}
	for _, name := range names[keep:] {
		_ = os.Remove(filepath.Join(dir, name))
	}
	return nil
}

func listHistory() []HistoryItem {
	state.Lock()
	dir := state.HistoryDir
	state.Unlock()
	files, err := os.ReadDir(dir)
	if err != nil {
		return []HistoryItem{}
	}
	var names []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			names = append(names, file.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	var items []HistoryItem
	if len(names) > 5 {
		names = names[:5]
	}
	for _, name := range names {
		raw, _ := os.ReadFile(filepath.Join(dir, name))
		items = append(items, HistoryItem{FileName: name, CreatedAt: name, ConfigRaw: string(raw)})
	}
	return items
}
