# IoT Dispense

IoT Dispense is the forwarding part of OT.io. It subscribes to OT.io MQTT telemetry and forwards matching messages to a target system.

The first implementation forwards to another MQTT broker. The module is intentionally shaped so other targets can be added later.

## Current target support

| Target type | Status |
|---|---|
| MQTT broker | Implemented in the beta demo. |
| Kafka | Planned. |
| NATS | Planned. |
| HTTP API | Planned. |
| Database sink | Planned. |
| Cloud target | Planned. |

## What Dispense does

1. subscribes to metric topics from the input broker
2. parses the OT.io metric message
3. filters by `sourceFilter`
4. builds a target topic under `targetPrefix`
5. publishes the original payload to the output broker
6. tracks received, forwarded and dropped messages

## Environment variables

| Variable | Meaning |
|---|---|
| `DISPENSE_INSTANCE_ID` | Instance id shown in Lense. |
| `DISPENSE_SOURCE_FILTER` | Source type to forward, for example `modbus-tcp` or `opcua`. |
| `DISPENSE_SOURCE_PREFIX` | Input MQTT topic prefix. |
| `DISPENSE_TARGET_PREFIX` | Output MQTT topic prefix. |
| `DISPENSE_BUFFER_LIMIT` | Number of metric points kept in memory. |
| `DISPENSE_INPUT_BROKER_HOST` | Input MQTT broker host. |
| `DISPENSE_INPUT_BROKER_PORT` | Input MQTT broker port. |
| `DISPENSE_INPUT_CLIENT_ID` | Input MQTT client id. |
| `DISPENSE_OUTPUT_BROKER_HOST` | Output MQTT broker host. |
| `DISPENSE_OUTPUT_BROKER_PORT` | Output MQTT broker port. |
| `DISPENSE_OUTPUT_CLIENT_ID` | Output MQTT client id. |
| `DISPENSE_UI_PORT` | UI/API port. |
| `DISPENSE_CONFIG_JSON` | Optional full JSON configuration. |

## Complete JSON configuration

```json
{
  "instanceId": "iot-dispense-opcua-01",
  "sourceFilter": "opcua",
  "sourcePrefix": "iot-lense",
  "targetPrefix": "dispense/opcua",
  "bufferLimit": 1500,
  "inputBroker": {
    "host": "mqtt",
    "port": 1883,
    "clientId": "iot-dispense-opcua-01-in"
  },
  "outputBroker": {
    "host": "dispense-broker",
    "port": 1883,
    "clientId": "iot-dispense-opcua-01-out"
  }
}
```

## Docker Compose example

```yaml
dispense-opcua:
  build:
    context: .
    dockerfile: apps/dispense/Dockerfile
  container_name: iot-dispense-opcua
  restart: unless-stopped
  depends_on:
    - mqtt
    - dispense-broker
  environment:
    DISPENSE_INSTANCE_ID: iot-dispense-opcua-01
    DISPENSE_SOURCE_FILTER: opcua
    DISPENSE_SOURCE_PREFIX: iot-lense
    DISPENSE_TARGET_PREFIX: dispense/opcua
    DISPENSE_INPUT_BROKER_HOST: mqtt
    DISPENSE_INPUT_BROKER_PORT: 1883
    DISPENSE_OUTPUT_BROKER_HOST: dispense-broker
    DISPENSE_OUTPUT_BROKER_PORT: 1883
    DISPENSE_UI_PORT: 8201
  ports:
    - "8201:8201"
```

## Forwarding behavior

If `sourceFilter` is `opcua`, only metric messages with `source.type = opcua` are forwarded.

The outgoing topic is built like this:

```text
<targetPrefix>/<agentId>/metrics/<metricName>
```

Example:

```text
dispense/opcua/opcua-machine-01/metrics/temperature
```

## Writing a custom Dispense extension

A Dispense extension is useful when OT.io data should be sent to another target system. The current beta includes MQTT-to-MQTT forwarding. The same pattern can be used for Kafka, NATS, HTTP APIs, databases or cloud services.

### Recommended folder layout

```text
apps/
└── dispense-mytarget/
    ├── Dockerfile
    ├── go.mod
    ├── main.go
    ├── config.go
    ├── filter.go
    ├── mapper.go
    ├── target_writer.go
    └── target_writer_test.go
```

### Minimal responsibilities

A custom Dispense module should:

1. load a JSON configuration
2. connect to the OT.io input transport
3. subscribe to telemetry topics
4. filter messages by source type, agent id or metric name
5. map OT.io messages to the target format
6. write to the target system
7. expose status, metrics and logs
8. publish or expose health information for Lense
9. include tests for filtering, mapping and target writing

### Recommended interfaces

Keep the target-specific code behind a small writer interface.

```go
type TargetWriter interface {
    Connect(ctx context.Context) error
    Write(ctx context.Context, message TargetMessage) error
    Close(ctx context.Context) error
}
```

The Dispense runtime should not need to know whether the target is MQTT, Kafka, NATS, HTTP or a database.

### Example target configuration

```json
{
  "http": {
    "listen": ":8210"
  },
  "inputBroker": {
    "clientId": "iot-dispense-http-01",
    "host": "mqtt",
    "port": 1883,
    "topicPrefix": "otio"
  },
  "sourceFilter": {
    "sourceType": "opcua",
    "includeAgentIds": [
      "opcua-machine-01"
    ],
    "excludeAgentIds": []
  },
  "target": {
    "type": "http",
    "url": "http://target-api:8080/telemetry",
    "method": "POST",
    "timeoutMs": 3000
  },
  "routes": [
    {
      "name": "opcua-to-http",
      "sourceTopic": "otio/+/metrics/#",
      "enabled": true
    }
  ]
}
```

### Example Docker Compose service

```yaml
dispense-http:
  build:
    context: .
    dockerfile: apps/dispense-http/Dockerfile
  container_name: iot-dispense-http
  environment:
    CONFIG_FILE: /app/config/config.json
  volumes:
    - ./apps/dispense-http/config/config.json:/app/config/config.json:ro
  ports:
    - "8210:8210"
  depends_on:
    - mqtt
```

### How Lense should know it

Add the module to the Lense topology configuration.

```json
{
  "id": "iot-dispense-http-01",
  "label": "IoT Dispense HTTP",
  "url": "http://127.0.0.1:8210",
  "sourceType": "opcua",
  "target": "http-api"
}
```

### Testing checklist

Before adding a Dispense extension to the demo stack:

- test configuration parsing
- test source filters
- test topic matching
- test mapping to target payload
- test retry/error behavior
- test health/status output
- test that Lense shows the module in the connection graph

## Delivery limits

`DISPENSE_BUFFER_LIMIT` limits in-memory chart history. It is not a durable forwarding queue. Input/output use MQTT QoS 0; disconnected output and failed publish attempts are counted as dropped or unconfirmed, with no persistent retry/replay. A timed-out publish can still complete later. Dispense is not included in the HA example.
