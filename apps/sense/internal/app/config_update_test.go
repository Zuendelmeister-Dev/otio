package app

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"iot-lense-sense/shared/mqttx"
)

func configUpdateFixture(t *testing.T, restart func()) string {
	t.Helper()
	t.Setenv("OTIO_CONFIG_WRITE_TOKEN", "")
	t.Setenv("OTIO_CONFIG_READ_ONLY", "")
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	raw, err := json.MarshalIndent(validConfig(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	normalized, err := normalizeJSON(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(normalized)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	previous, previousRestart := state, restartWorkersAfterConfigApply
	state = &AppState{Config: validConfig(), ConfigRaw: string(raw), ConfigPath: path, HistoryDir: filepath.Join(dir, "history"), ConfigLoaded: true, ConfigValid: true, Proposals: map[string]Proposal{}, Sources: map[string]SourceStatus{}}
	restartWorkersAfterConfigApply = restart
	t.Cleanup(func() {
		waitForConfigRestart(t)
		state, restartWorkersAfterConfigApply = previous, previousRestart
	})
	return string(raw)
}

func proposeConfig(t *testing.T, raw string) Proposal {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"raw": raw})
	w := httptest.NewRecorder()
	apiValidateConfig(w, httptest.NewRequest(http.MethodPost, "/api/config/validate", strings.NewReader(string(body))))
	var result struct {
		Valid    bool     `json:"valid"`
		Proposal Proposal `json:"proposal"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || !result.Valid {
		t.Fatalf("validation failed: %s (%v)", w.Body.String(), err)
	}
	return result.Proposal
}

func applyProposal(id string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"proposalId": id})
	w := httptest.NewRecorder()
	apiApplyConfig(w, httptest.NewRequest(http.MethodPost, "/api/config/apply", strings.NewReader(string(body))))
	return w
}

func configStatus(t *testing.T) ConfigUpdate {
	t.Helper()
	w := httptest.NewRecorder()
	apiStatus(w, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	var status struct {
		Config struct {
			Update ConfigUpdate `json:"update"`
		} `json:"config"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	return status.Config.Update
}

func TestApplyReturnsBeforeBrokerStartupAndRejectsConcurrentChanges(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	oldRaw := configUpdateFixture(t, func() { close(entered); <-release })
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	newRaw := strings.Replace(oldRaw, `"clientId": "sense"`, `"clientId": "next-sense"`, 1)
	proposal := proposeConfig(t, newRaw)
	if proposal.BaseRevision != configRevision(oldRaw) {
		t.Fatal("proposal is not bound to the validated base configuration")
	}
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- applyProposal(proposal.ID) }()
	var response *httptest.ResponseRecorder
	select {
	case response = <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("apply waited for the broker instead of acknowledging the saved file")
	}
	if response.Code != http.StatusAccepted || !strings.Contains(response.Body.String(), `"saved":true`) {
		t.Fatalf("expected saved/restarting response: %d %s", response.Code, response.Body.String())
	}
	<-entered
	if update := configStatus(t); update.Phase != "restarting" || update.Revision != configRevision(newRaw) {
		t.Fatalf("unexpected restart status: %+v", update)
	}
	second := proposeConfig(t, oldRaw)
	if w := applyProposal(second.ID); w.Code != http.StatusConflict {
		t.Fatalf("concurrent apply accepted: %d %s", w.Code, w.Body.String())
	}
	rollback := httptest.NewRecorder()
	apiRollbackConfig(rollback, httptest.NewRequest(http.MethodPost, "/api/config/rollback", strings.NewReader(`{"fileName":"previous.json"}`)))
	if rollback.Code != http.StatusConflict {
		t.Fatalf("concurrent rollback accepted: %d", rollback.Code)
	}
	state.Lock()
	path := state.ConfigPath
	state.Unlock()
	written, err := os.ReadFile(path)
	if err != nil || string(written) != newRaw {
		t.Fatalf("saved configuration differs: %q %v", written, err)
	}
	releaseOnce.Do(func() { close(release) })
	waitForConfigRestart(t)
	if update := configStatus(t); update.Phase != "connecting" {
		t.Fatalf("unavailable broker should remain connecting: %+v", update)
	}
	state.Lock()
	state.BrokerConnected = true
	state.Unlock()
	if update := configStatus(t); update.Phase != "ready" {
		t.Fatalf("later reconnect was not reflected: %+v", update)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".otio-config-") {
			t.Fatal("temporary configuration file was left behind")
		}
	}
}

func TestStaleProposalCannotOverwriteNewerConfiguration(t *testing.T) {
	oldRaw := configUpdateFixture(t, func() {})
	first := proposeConfig(t, strings.Replace(oldRaw, `"clientId": "sense"`, `"clientId": "first"`, 1))
	stale := proposeConfig(t, strings.Replace(oldRaw, `"clientId": "sense"`, `"clientId": "stale"`, 1))
	if w := applyProposal(first.ID); w.Code != http.StatusAccepted {
		t.Fatal(w.Body.String())
	}
	waitForConfigRestart(t)
	if w := applyProposal(stale.ID); w.Code != http.StatusConflict {
		t.Fatalf("stale proposal accepted: %d %s", w.Code, w.Body.String())
	}
	state.Lock()
	clientID := state.Config.Broker.ClientID
	state.Unlock()
	if clientID != "first" || len(listHistory()) != 1 {
		t.Fatalf("stale proposal changed active config or history: %s", clientID)
	}
}

func TestSavedConfigurationSurvivesBrokerThatNeverAcknowledgesConnect(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()
	var brokerConn net.Conn
	defer func() {
		if brokerConn != nil {
			brokerConn.Close()
		}
	}()
	oldRaw := configUpdateFixture(t, func() {
		state.Lock()
		config := state.Config
		state.Unlock()
		connectTransport(config)
	})
	t.Cleanup(func() {
		if brokerConn != nil {
			brokerConn.Close()
		}
		waitForConfigRestart(t)
		state.Lock()
		client := state.MQTTClient
		state.Unlock()
		if client != nil {
			client.Disconnect(0)
		}
	})
	config := validConfig()
	config.Broker.Host = "127.0.0.1"
	config.Broker.Port = listener.Addr().(*net.TCPAddr).Port
	newRaw, _ := json.Marshal(config)
	proposal := proposeConfig(t, string(newRaw))
	start := time.Now()
	response := applyProposal(proposal.ID)
	if response.Code != http.StatusAccepted || time.Since(start) >= time.Second {
		t.Fatalf("saving was blocked by MQTT startup: %d %s", response.Code, response.Body.String())
	}
	select {
	case brokerConn = <-accepted:
	case <-time.After(time.Second):
		t.Fatal("collector did not attempt the broker connection")
	}
	if update := configStatus(t); update.Phase != "restarting" {
		t.Fatalf("connection attempt should still be in progress: %+v", update)
	}
	// The broker accepts TCP but sends no CONNACK. The five-second startup
	// timeout must leave the saved configuration intact and expose connecting.
	brokerConn.Close()
	waitForConfigRestart(t)
	if update := configStatus(t); update.Phase != "connecting" || update.Revision == configRevision(oldRaw) {
		t.Fatalf("saved configuration lost after connection failure: %+v", update)
	}
	state.Lock()
	path, client := state.ConfigPath, state.MQTTClient
	state.Unlock()
	diskRaw, err := os.ReadFile(path)
	if err != nil || configRevision(string(diskRaw)) != configRevision(proposal.ConfigRaw) || mqttx.IsReady(client) {
		t.Fatal("broker failure changed the saved configuration or reports ready")
	}
}

func TestExternalFileChangeRejectsPreviouslyValidatedProposal(t *testing.T) {
	oldRaw := configUpdateFixture(t, func() { t.Error("restarted an uncommitted configuration") })
	proposal := proposeConfig(t, oldRaw)
	state.Lock()
	path := state.ConfigPath
	state.Unlock()
	external := strings.Replace(oldRaw, `"clientId": "sense"`, `"clientId": "external"`, 1)
	if err := os.WriteFile(path, []byte(external), 0600); err != nil {
		t.Fatal(err)
	}
	if w := applyProposal(proposal.ID); w.Code != http.StatusConflict {
		t.Fatalf("external file edit overwritten: %d %s", w.Code, w.Body.String())
	}
	written, _ := os.ReadFile(path)
	if string(written) != external || len(listHistory()) != 0 {
		t.Fatal("conflict changed disk configuration or added misleading history")
	}
}

func TestRollbackUsesAsyncRestartAndUpdatesRevision(t *testing.T) {
	oldRaw := configUpdateFixture(t, func() {})
	proposal := proposeConfig(t, strings.Replace(oldRaw, `"clientId": "sense"`, `"clientId": "next"`, 1))
	if w := applyProposal(proposal.ID); w.Code != http.StatusAccepted {
		t.Fatal(w.Body.String())
	}
	waitForConfigRestart(t)
	history := listHistory()
	body, _ := json.Marshal(map[string]string{"fileName": history[0].FileName})
	w := httptest.NewRecorder()
	apiRollbackConfig(w, httptest.NewRequest(http.MethodPost, "/api/config/rollback", strings.NewReader(string(body))))
	if w.Code != http.StatusAccepted {
		t.Fatalf("rollback failed: %d %s", w.Code, w.Body.String())
	}
	waitForConfigRestart(t)
	if update := configStatus(t); update.Revision != configRevision(oldRaw) {
		t.Fatalf("rollback revision differs: %+v", update)
	}
}
