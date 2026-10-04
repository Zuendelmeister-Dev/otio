package main

import "iot-lense-sense/shared/protocolmeta"

type Capability = protocolmeta.Capability

var catalog = protocolmeta.Lab()

func capability(id string) (Capability, bool) {
	for _, c := range catalog {
		if c.ID == id {
			return c, true
		}
	}
	return Capability{}, false
}
