package main

import "sync"

type AppState struct {
	sync.Mutex
	BrokerConnected bool
	LastMessage     string
	MessageCount    int64
	Topics          map[string]bool
	ComponentStatus map[string]ComponentStatus
}

var state = &AppState{Topics: map[string]bool{}, ComponentStatus: map[string]ComponentStatus{}}
