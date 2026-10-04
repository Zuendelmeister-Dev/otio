// Package protocolmeta describes the implemented protocol capabilities and
// configuration presets shared by Sense, Protocol Lab and the form editor.
// Driver implementations remain in their respective applications.
package protocolmeta

import (
	_ "embed"
	"encoding/json"
)

//go:embed catalog.json
var catalogJSON []byte

type MetricPreset struct {
	Register *uint16 `json:"register,omitempty"`
	NodeID   string  `json:"nodeId,omitempty"`
	Address  string  `json:"address,omitempty"`
	Scale    float64 `json:"scale"`
}

type SourcePreset struct {
	Type               string       `json:"type"`
	Label              string       `json:"label"`
	Port               int          `json:"port"`
	Connection         string       `json:"connection,omitempty"`
	Metric             MetricPreset `json:"metric"`
	AddressField       string       `json:"addressField"`
	RequiresUnitID     bool         `json:"requiresUnitId,omitempty"`
	RequiresConnection bool         `json:"requiresConnection,omitempty"`
	Poll               bool         `json:"poll"`
	Subscription       bool         `json:"subscription"`
}

type SimulatorPreset struct {
	Port       int     `json:"port"`
	Connection string  `json:"connection"`
	Address    string  `json:"address"`
	Scale      float64 `json:"scale"`
	Notes      string  `json:"notes"`
}

// Capability distinguishes native reads and samples from hardware integrations
// that are documented but not implemented. Sources lists usable Sense types;
// it never turns a sampled Protocol Lab read into a continuous subscription.
type Capability struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Transport       string           `json:"transport"`
	Status          string           `json:"status"`
	Notes           string           `json:"notes"`
	Connection      string           `json:"connection"`
	Address         string           `json:"address"`
	Scheme          string           `json:"scheme,omitempty"`
	Lab             bool             `json:"lab"`
	LabRead         bool             `json:"labRead"`
	Simulator       bool             `json:"simulator"`
	SimulatorPreset *SimulatorPreset `json:"simulatorPreset,omitempty"`
	Sources         []SourcePreset   `json:"sources,omitempty"`
}

// All returns fresh metadata so application code cannot alter the catalog used
// by another consumer. The catalog is small and is read only during setup.
func All() []Capability {
	var result []Capability
	if err := json.Unmarshal(catalogJSON, &result); err != nil {
		panic("invalid embedded protocol catalog: " + err.Error())
	}
	return result
}

func Lab() []Capability {
	result := []Capability{}
	for _, item := range All() {
		if item.Lab {
			result = append(result, item)
		}
	}
	return result
}

func Sources() []SourcePreset {
	result := []SourcePreset{}
	for _, item := range All() {
		result = append(result, item.Sources...)
	}
	return result
}

func Source(sourceType string) (SourcePreset, bool) {
	for _, item := range Sources() {
		if item.Type == sourceType {
			return item, true
		}
	}
	return SourcePreset{}, false
}
