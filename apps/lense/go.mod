module iot-lense-sense/lense

go 1.26.0

require (
	github.com/lib/pq v1.10.9
	iot-lense-sense/shared/mqttx v0.0.0
)

require (
	github.com/eclipse/paho.mqtt.golang v1.5.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
)

replace iot-lense-sense/shared/mqttx => ../../shared/mqttx
