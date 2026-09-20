package main

import (
	plc "github.com/apache/plc4x/plc4go/pkg/api"
	"github.com/apache/plc4x/plc4go/pkg/api/config"
	"github.com/apache/plc4x/plc4go/pkg/api/drivers"
	"github.com/rs/zerolog"
	"testing"
)

func TestCatalogAddressesMatchPinnedDrivers(t *testing.T) {
	quiet := config.WithCustomLogger(zerolog.Nop())
	manager := plc.NewPlcDriverManager(quiet)
	defer manager.Close()
	registered := map[string]plc.PlcDriver{
		"s7":              drivers.RegisterS7Driver(manager, quiet),
		"ethernet-ip":     drivers.RegisterEipDriver(manager, quiet),
		"bacnet-ip":       drivers.RegisterBacnetDriver(manager, quiet),
		"knxnet-ip":       drivers.RegisterKnxDriver(manager, quiet),
		"iec-60870-5-104": drivers.RegisterIec608705104Driver(manager, quiet),
	}
	for protocol, driver := range registered {
		t.Run(protocol, func(t *testing.T) {
			c, _ := capability(protocol)
			if err := driver.CheckTagAddress(c.Address); err != nil {
				t.Fatalf("invalid default address %s: %v", c.Address, err)
			}
		})
	}
}
