package mqttx

import mqtt "github.com/eclipse/paho.mqtt.golang"

// Client is the MQTT client abstraction exposed to OT.io modules.
type Client = mqtt.Client

// Message is the MQTT message abstraction exposed to OT.io modules.
type Message = mqtt.Message
