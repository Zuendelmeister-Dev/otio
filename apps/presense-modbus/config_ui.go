package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type simulatorSettings struct {
	Protocol      string  `json:"protocol"`
	GeneratorMode string  `json:"generatorMode"`
	Temperature   float64 `json:"temperature"`
	Humidity      float64 `json:"humidity"`
	Pressure      float64 `json:"pressure"`
	Vibration     float64 `json:"vibration"`
	Speed         float64 `json:"speed"`
	Current       float64 `json:"current"`
}

var simulatorConfig struct {
	sync.RWMutex
	value *simulatorSettings
}

func simulatorMode(fallback string) string {
	if s := currentSimulatorSettings(); s != nil {
		return s.GeneratorMode
	}
	return fallback
}
func currentSimulatorSettings() *simulatorSettings {
	simulatorConfig.RLock()
	defer simulatorConfig.RUnlock()
	if simulatorConfig.value == nil {
		return nil
	}
	copy := *simulatorConfig.value
	return &copy
}
func validateSimulatorSettings(s simulatorSettings, protocol string) error {
	if s.Protocol != protocol {
		return errors.New("this endpoint has a fixed wire protocol; choose another simulator in Protocol Lab")
	}
	modes := map[string]bool{"constant": true, "sine": true, "random-int": true, "random-string": true, "array": true, "sawtooth": true, "square": true, "triangle": true}
	if !modes[s.GeneratorMode] {
		return errors.New("unsupported generator mode")
	}
	for _, v := range []float64{s.Temperature, s.Humidity, s.Pressure, s.Vibration, s.Speed, s.Current} {
		if v < 0 || v > 6000 {
			return errors.New("base values must be between 0 and 6000")
		}
	}
	return nil
}
func initSimulatorSettings(s simulatorSettings) {
	if path := os.Getenv("PRESENSE_CONFIG_PATH"); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			var saved simulatorSettings
			if json.Unmarshal(data, &saved) == nil && validateSimulatorSettings(saved, s.Protocol) == nil {
				s = saved
			}
		}
	}
	simulatorConfig.Lock()
	simulatorConfig.value = &s
	simulatorConfig.Unlock()
}
func simulatorSource(s simulatorSettings, id, host string, port int) map[string]any {
	metrics := []map[string]any{}
	if s.Protocol == "modbus-tcp" {
		for i, name := range []string{"temperature", "humidity", "pressure", "vibration", "status", "cycle"} {
			scale := 1.0
			if i < 2 {
				scale = 0.1
			} else if i < 4 {
				scale = 0.01
			}
			metrics = append(metrics, map[string]any{"name": name, "register": i, "scale": scale})
		}
	} else {
		for _, name := range []string{"Temperature", "Speed", "Current", "Load", "State"} {
			metrics = append(metrics, map[string]any{"name": name, "nodeId": "ns=2;s=Machine." + name, "scale": 1})
		}
	}
	return map[string]any{"agentId": id, "type": s.Protocol, "host": host, "port": port, "unitId": 1, "metrics": metrics}
}
func registerSimulatorConfig(mux *http.ServeMux, id, host string, port int) {
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		allowed := map[string]bool{"workspace-ui.js": true, "config-form.js": true, "presense-ui.js": true, "lab-link.js": true}
		if !allowed[name] {
			http.NotFound(w, r)
			return
		}
		for _, dir := range []string{"/app/static", "../../shared/web", "shared/web"} {
			if data, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
				w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
				_, _ = w.Write(data)
				return
			}
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPut {
			if token := os.Getenv("OTIO_CONFIG_WRITE_TOKEN"); token != "" && r.Header.Get("X-OTIO-Config-Token") != token {
				http.Error(w, "configuration token required", http.StatusUnauthorized)
				return
			}
			var next simulatorSettings
			d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
			d.DisallowUnknownFields()
			if err := d.Decode(&next); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			var extra any
			if err := d.Decode(&extra); err != io.EOF {
				http.Error(w, "expected one JSON object", 400)
				return
			}
			current := currentSimulatorSettings()
			if current == nil {
				http.Error(w, "simulator not initialized", 503)
				return
			}
			if err := validateSimulatorSettings(next, current.Protocol); err != nil {
				http.Error(w, err.Error(), 422)
				return
			}
			simulatorConfig.Lock()
			if path := os.Getenv("PRESENSE_CONFIG_PATH"); path != "" {
				data, _ := json.MarshalIndent(next, "", "  ")
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					simulatorConfig.Unlock()
					http.Error(w, err.Error(), 500)
					return
				}
				if err := saveSimulatorSettings(path, data); err != nil {
					simulatorConfig.Unlock()
					http.Error(w, err.Error(), 500)
					return
				}
			}
			simulatorConfig.value = &next
			simulatorConfig.Unlock()
		} else if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET, PUT")
			http.Error(w, "method not allowed", 405)
			return
		}
		settings := currentSimulatorSettings()
		if settings == nil {
			http.Error(w, "simulator not initialized", 503)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"config": settings, "source": simulatorSource(*settings, id, host, port), "persistent": os.Getenv("PRESENSE_CONFIG_PATH") != ""})
	})
}

// Replace the complete snapshot only after a successful write and flush.
func saveSimulatorSettings(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".simulator-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
