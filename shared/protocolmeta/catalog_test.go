package protocolmeta

import (
	"strings"
	"testing"
)

func TestCatalogCapabilitiesAreConsistent(t *testing.T) {
	ids, types := map[string]bool{}, map[string]bool{}
	for _, item := range All() {
		if item.ID == "" || ids[item.ID] {
			t.Fatalf("duplicate or empty protocol id %q", item.ID)
		}
		ids[item.ID] = true
		if item.LabRead && (!item.Lab || item.Scheme == "" || !strings.HasPrefix(item.Connection, item.Scheme+"://") || item.Address == "") {
			t.Fatalf("unusable native read preset: %+v", item)
		}
		if item.Simulator != (item.SimulatorPreset != nil) {
			t.Fatalf("simulator metadata does not match availability: %s", item.ID)
		}
		if item.SimulatorPreset != nil && (item.SimulatorPreset.Port < 1 || item.SimulatorPreset.Port > 65535 || item.SimulatorPreset.Scale == 0 || item.SimulatorPreset.Address == "") {
			t.Fatalf("invalid simulator preset: %s", item.ID)
		}
		labSource := false
		for _, source := range item.Sources {
			if source.Type == "" || types[source.Type] || source.Port < 1 || source.Port > 65535 || source.Metric.Scale == 0 || (!source.Poll && !source.Subscription) {
				t.Fatalf("invalid or duplicate source preset: %+v", source)
			}
			types[source.Type] = true
			switch source.AddressField {
			case "register", "nodeId", "address":
			default:
				t.Fatalf("unsupported address field for %s", source.Type)
			}
			if source.RequiresConnection {
				labSource = true
				if !item.LabRead || source.Type != "lab-"+item.ID || source.Connection == "" || source.Subscription {
					t.Fatalf("native Lab read must advertise a usable polling collector: %s", source.Type)
				}
			}
		}
		if item.LabRead != labSource {
			t.Fatalf("Lab read and Sense collector diverge for %s", item.ID)
		}
	}
	if len(Lab()) == 0 || len(Sources()) == 0 {
		t.Fatal("empty protocol catalog")
	}
}

func TestCatalogAccessDoesNotShareMutableMetadata(t *testing.T) {
	items := All()
	items[0].Name = "modified"
	items[0].Sources[0].Type = "modified"
	if All()[0].Name == "modified" || All()[0].Sources[0].Type == "modified" {
		t.Fatal("a consumer changed the canonical catalog")
	}
	if _, ok := Source("lab-mbus-tcp"); !ok {
		t.Fatal("the native M-Bus read lacks its Sense collector preset")
	}
	if _, ok := Source("unimplemented-protocol"); ok {
		t.Fatal("unknown source advertised as implemented")
	}
}
