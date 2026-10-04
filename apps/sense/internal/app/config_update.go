package app

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Serialize file commits without holding the status mutex during filesystem IO.
// The asynchronous restart is protected by ConfigUpdate.Phase after this lock
// is released, so a second request cannot replace a restart's configuration.
var configMutation sync.Mutex

var errConfigConflict = errors.New("configuration changed; reload and validate again")

func configRevision(raw string) string {
	if normalized, err := normalizeJSON(raw); err == nil {
		raw = normalized
	}
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

// configUpdateLocked must be called with state locked. Readiness follows the
// connection, including a reconnect that succeeds after the initial timeout.
func configUpdateLocked() ConfigUpdate {
	update := state.ConfigUpdate
	update.Revision = configRevision(state.ConfigRaw)
	if update.Phase != "restarting" {
		if state.BrokerConnected {
			update.Phase = "ready"
			update.Message = "Configuration is active and the broker is connected."
		} else {
			update.Phase = "connecting"
			update.Message = "Configuration is saved; waiting for the broker connection."
		}
	}
	return update
}

// atomicWriteConfig creates the replacement in the same directory. Readers
// observe either the previous complete JSON file or the new complete file.
func atomicWriteConfig(path string, raw []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".otio-config-*.tmp")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if err = file.Chmod(info.Mode().Perm()); err == nil {
		_, err = file.Write(raw)
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

// commitConfig assumes configMutation is held and returns before broker IO.
func commitConfig(raw, expectedRevision string) (string, ConfigUpdate, error) {
	config, problems := parseAndValidate(raw)
	if len(problems) != 0 {
		return "", ConfigUpdate{}, fmt.Errorf("invalid configuration: %v", problems)
	}
	state.Lock()
	path, oldRaw := state.ConfigPath, state.ConfigRaw
	currentRevision := configRevision(oldRaw)
	state.Unlock()
	if expectedRevision != currentRevision {
		return "", ConfigUpdate{}, errConfigConflict
	}
	// Refuse to overwrite a configuration changed outside this API as well.
	diskRaw, err := os.ReadFile(path)
	if err != nil {
		return "", ConfigUpdate{}, err
	}
	if configRevision(string(diskRaw)) != currentRevision {
		return "", ConfigUpdate{}, errConfigConflict
	}
	historyFile, err := saveHistorySnapshot(oldRaw)
	if err != nil {
		return "", ConfigUpdate{}, err
	}
	if err := atomicWriteConfig(path, []byte(raw)); err != nil {
		return "", ConfigUpdate{}, err
	}
	update := ConfigUpdate{Phase: "restarting", Revision: configRevision(raw), StartedAt: nowISO(), Message: "Configuration is saved; workers are restarting."}
	state.Lock()
	state.Config = config
	state.ConfigRaw = raw
	state.ConfigRevision = update.Revision
	state.ConfigLoaded = true
	state.ConfigValid = true
	state.ConfigError = ""
	state.ConfigUpdate = update
	state.BrokerConnected = false
	state.Unlock()
	// Capture the function before starting the goroutine (also a useful test seam).
	restart := restartWorkersAfterConfigApply
	go func() {
		restart()
		state.Lock()
		state.ConfigUpdate.Phase = "connecting"
		state.ConfigUpdate.CompletedAt = nowISO()
		state.ConfigUpdate = configUpdateLocked()
		state.Unlock()
	}()
	return historyFile, update, nil
}
