module iot-lense-sense/lense

go 1.22

require (
	iot-lense-sense/shared/mqttx v0.0.0
	github.com/lib/pq v1.10.9
)

replace iot-lense-sense/shared/mqttx => ../../shared/mqttx
