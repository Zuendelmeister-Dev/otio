package main

import (
	"context"
	"fmt"
	"os"
	"sync"
)

// Listeners are opt-in. Diagnostic reads never start a simulator.
type simulationRuntime struct {
	mu     sync.Mutex
	ctx    context.Context
	sim    *simulator
	active map[string]func()
}

func newSimulationRuntime(ctx context.Context, sim *simulator) *simulationRuntime {
	return &simulationRuntime{ctx: ctx, sim: sim, active: map[string]func(){}}
}
func (s *simulationRuntime) names() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	names := []string{}
	for name := range s.active {
		names = append(names, name)
	}
	return names
}
func (s *simulationRuntime) stopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, stop := range s.active {
		stop()
		delete(s.active, name)
	}
}
func (s *simulationRuntime) set(protocol string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if stop, ok := s.active[protocol]; ok {
		if !enabled {
			stop()
			delete(s.active, protocol)
		}
		return nil
	}
	supported := map[string]bool{"modbus-tcp": true, "modbus-rtu-tcp": true, "opcua-tcp": true, "mqtt": true, "amqp": true, "s7": true, "mbus-tcp": true}
	if !supported[protocol] {
		return fmt.Errorf("no simulator for %s", protocol)
	}
	if !enabled {
		return nil
	}
	if protocol == "mqtt" && os.Getenv("LAB_MQTT_URL") == "" {
		return fmt.Errorf("configure LAB_MQTT_URL before starting the MQTT publisher")
	}
	if protocol == "amqp" && os.Getenv("LAB_AMQP_URL") == "" {
		return fmt.Errorf("configure LAB_AMQP_URL before starting the AMQP publisher")
	}
	ctx, cancel := context.WithCancel(s.ctx)
	var closeListener func()
	switch protocol {
	case "modbus-tcp", "modbus-rtu-tcp":
		port := 1502
		if protocol == "modbus-rtu-tcp" {
			port = 1503
		}
		l, err := listenModbus(ctx, s.sim, port, protocol == "modbus-rtu-tcp")
		if err != nil {
			cancel()
			return err
		}
		closeListener = func() { _ = l.Close() }
	case "opcua-tcp":
		server, err := startOPCUA(ctx, s.sim, "0.0.0.0", 4842)
		if err != nil {
			cancel()
			return err
		}
		closeListener = func() { _ = server.Close() }
	case "s7", "mbus-tcp":
		l, err := listenTCPDemo(ctx, s.sim, protocol)
		if err != nil {
			cancel()
			return err
		}
		closeListener = func() { _ = l.Close() }
	default:
		go publishDemoProtocol(ctx, s.sim, protocol)
	}
	s.active[protocol] = func() {
		cancel()
		if closeListener != nil {
			closeListener()
		}
	}
	return nil
}
