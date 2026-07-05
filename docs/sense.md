# IoT Sense

IoT Sense is the collector part of OT.io. It reads source endpoints and publishes telemetry, status, health and errors to MQTT.

A Sense instance is usually specialized for one source type, for example one Modbus collector and one OPC UA collector.

## Supported source types

| Source type | Supported read modes | Description |
|---|---|---|
| `modbus-tcp` | `poll` | Reads Modbus holding registers. |
| `opcua` | `poll` | Reads OPC UA style nodes through `/read`. |
| `opcua` | `subscription` | Subscribes to OPC UA style node updates through `/subscribe`. |

## MQTT output

Sense publishes:

| Message type | Topic pattern |
|---|---|
| Metric | `<topicPrefix>/<agentId>/metrics/<metricName>` |
| Status | `<topicPrefix>/<agentId>/status` |
| Error | `<topicPrefix>/<agentId>/errors` |
| Sense status | `<topicPrefix>/_sense/status` |

The default demo prefix is `iot-lense`.

## Configuration file

Sense reads `/app/config/config.json` in the container.

In Docker Compose, the config directories are mounted like this:

```yaml
volumes:
  - ./apps/sense/config-opcua:/app/config
```

## Top-level configuration fields

| Field | Required | Meaning |
|---|---:|---|
| `broker.host` | yes | MQTT broker host. |
| `broker.port` | yes | MQTT broker port. |
| `broker.clientId` | yes | MQTT client id. |
| `broker.topicPrefix` | yes | MQTT topic prefix. |
| `pollIntervalMs` | yes | Polling interval in milliseconds. Also used as fallback subscription interval. |
| `healthTimeoutSeconds` | no | Timeout for health views. Defaults to `300` where needed. |
| `sources` | yes | List of source endpoints. |

## Source configuration fields

| Field | Source type | Meaning |
|---|---|---|
| `agentId` | all | Source id used in topics and shown in Lense. |
| `type` | all | `modbus-tcp` or `opcua`. |
| `host` | all | Source host. In Compose this is usually the service name. |
| `port` | all | Source port. |
| `unitId` | Modbus | Modbus unit id. Must be greater than 0 for Modbus. |
| `readMode` | all | `poll` or `subscription`. Default is `poll`. Subscription is currently OPC UA only. |
| `subscriptionIntervalMs` | OPC UA subscription | Subscription update interval. Minimum is 250 ms. |
| `metrics` | all | List of metrics or node values to read. |

## Metric configuration fields

| Field | Source type | Meaning |
|---|---|---|
| `name` | all | Metric name used in MQTT topics. |
| `register` | Modbus | Holding register address. |
| `nodeId` | OPC UA | OPC UA node id. |
| `scale` | numeric values | Numeric scaling factor. Must not be 0. |
| `unit` | all | Display unit. |
| `type` | all | `gauge`, `string` or another descriptive type. |

Numeric metric values are charted. String and array-like values are shown as tables in Lense.

## Complete Modbus example

```json
{
  "broker": {
    "host": "mqtt",
    "port": 1883,
    "clientId": "iot-sense-modbus-01",
    "topicPrefix": "iot-lense"
  },
  "pollIntervalMs": 1000,
  "healthTimeoutSeconds": 300,
  "sources": [
    {
      "agentId": "modbus-machine-01",
      "type": "modbus-tcp",
      "host": "presense-modbus-01",
      "port": 5020,
      "unitId": 1,
      "readMode": "poll",
      "metrics": [
        {
          "name": "temperature",
          "register": 0,
          "scale": 0.1,
          "unit": "°C",
          "type": "gauge"
        },
        {
          "name": "humidity",
          "register": 1,
          "scale": 0.1,
          "unit": "%",
          "type": "gauge"
        }
      ]
    }
  ]
}
```

## Complete OPC UA subscription example

```json
{
  "broker": {
    "host": "mqtt",
    "port": 1883,
    "clientId": "iot-sense-opcua-01",
    "topicPrefix": "iot-lense"
  },
  "pollIntervalMs": 1000,
  "healthTimeoutSeconds": 300,
  "sources": [
    {
      "agentId": "opcua-machine-01",
      "type": "opcua",
      "host": "presense-opcua-01",
      "port": 4840,
      "unitId": 0,
      "readMode": "subscription",
      "subscriptionIntervalMs": 1000,
      "metrics": [
        {
          "name": "temperature",
          "nodeId": "ns=2;s=Machine.Temperature",
          "scale": 1,
          "unit": "°C",
          "type": "gauge"
        },
        {
          "name": "state",
          "nodeId": "ns=2;s=Machine.State",
          "scale": 1,
          "unit": "",
          "type": "string"
        }
      ]
    }
  ]
}
```

## Connecting a real machine

Replace the Presense host and port with the machine endpoint.

Example Modbus source:

```json
{
  "agentId": "real-press-01",
  "type": "modbus-tcp",
  "host": "real-press-01.local",
  "port": 502,
  "unitId": 1,
  "readMode": "poll",
  "metrics": [
    {
      "name": "pressure",
      "register": 2,
      "scale": 0.01,
      "unit": "bar",
      "type": "gauge"
    }
  ]
}
```

After changing Sense sources, update Lense topology so the connection graph knows where the source belongs.

## Writing a custom Sense extension

A Sense extension is the most important extension point when a new source protocol must be integrated. This is the preferred place for protocol-specific code. In real environments this will usually be required because every machine landscape has its own protocol mix, data model and quirks.

A Sense module should translate one source protocol into the common OT.io message model.

### Recommended folder layout

```text
apps/
└── sense-myprotocol/
    ├── Dockerfile
    ├── go.mod
    ├── main.go
    ├── config.go
    ├── reader.go
    ├── mapper.go
    ├── status.go
    ├── reader_test.go
    └── config_test.go
```

If the protocol can be implemented as another reader inside the existing Sense app, prefer a smaller layout:

```text
apps/
└── sense/
    ├── myprotocol_reader.go
    ├── myprotocol_config.go
    └── myprotocol_reader_test.go
```

### Minimal responsibilities

A custom Sense extension should:

1. load and validate its JSON configuration
2. connect to the source endpoint
3. read or subscribe to source values
4. map raw source values to OT.io metric messages
5. publish telemetry through the shared transport package
6. publish status, health and error messages
7. expose `/api/status`, logs and basic UI information
8. make its sources visible to Lense
9. include tests for config parsing, reading/subscribing and mapping

### Recommended reader interface

Keep protocol-specific code behind a small reader interface.

```go
type SourceReader interface {
    Connect(ctx context.Context) error
    Read(ctx context.Context) ([]MetricValue, error)
    Subscribe(ctx context.Context, handler func(MetricValue)) error
    Close(ctx context.Context) error
}
```

Polling-only protocols can implement `Read`. Subscription-capable protocols can implement `Subscribe`. If a protocol does not support subscriptions, return a clear unsupported error.

### Example source configuration

```json
{
  "agentId": "my-machine-01",
  "displayName": "My Machine 01",
  "type": "myprotocol",
  "origin": "external",
      "host": "custom-sensor-01.local",
  "port": 1234,
  "readMode": "poll",
  "pollIntervalMs": 1000,
  "metrics": [
    {
      "name": "temperature",
      "type": "gauge",
      "unit": "°C",
      "path": "process.temperature",
      "scale": 1
    },
    {
      "name": "machineState",
      "type": "string",
      "unit": "",
      "path": "status.state"
    }
  ]
}
```

### Source origin

Use `origin` to tell Lense how to render the source endpoint.

| Value | Meaning |
|---|---|
| `presense` | The source is an OT.io simulator. It is shown as a dashed clickable simulation node. |
| `external` | The source is a real machine or external endpoint. It is shown as a solid non-clickable node. |

### Mapping rules

The Sense extension should normalize protocol-specific values into simple OT.io metric values.

| Raw source value | OT.io behavior |
|---|---|
| number | charted as a numeric metric |
| string | shown as a table value in Lense |
| boolean | can be published as boolean or mapped to a string state |
| array/object | shown as table/json-like value unless a custom mapper converts it |

### MQTT publishing

Do not implement MQTT plumbing inside the extension. Use the shared transport package.

The extension should only decide:

- which values to read
- how to name metrics
- how to map values
- when to mark a source healthy or unhealthy

The shared transport layer should handle:

- broker connection
- topic formatting
- JSON publishing
- reconnect behavior
- shared payload conventions

### Example Docker Compose service

```yaml
sense-myprotocol:
  build:
    context: .
    dockerfile: apps/sense-myprotocol/Dockerfile
  container_name: iot-sense-myprotocol
  environment:
    CONFIG_FILE: /app/config/config.json
  volumes:
    - ./apps/sense-myprotocol/config/config.json:/app/config/config.json:ro
  ports:
    - "8110:8100"
  depends_on:
    - mqtt
```

### How Lense should know it

Add the Sense instance to the Lense topology configuration.

```json
{
  "id": "iot-sense-myprotocol-01",
  "label": "IoT Sense MyProtocol",
  "url": "http://127.0.0.1:8110",
  "sourceType": "myprotocol"
}
```

### Testing checklist

Before adding a Sense extension to the demo stack:

- test configuration validation
- test connection failure handling
- test poll mode
- test subscription mode if supported
- test value mapping for numbers, strings and arrays
- test health state changes
- test MQTT message creation through shared transport
- test Lense graph behavior with the new source type


## Extension readiness review

The current Sense code is structured so that protocol-specific code is isolated.

| Concern | Location | Should a protocol extension touch it? |
|---|---|---|
| HTTP API | `apps/sense/internal/app` | Usually no |
| UI serving | `apps/sense/internal/app` and `apps/sense/static` | Usually no |
| Configuration workflow | `apps/sense/internal/app` | Usually no |
| Worker lifecycle | `apps/sense/internal/app` | Usually no |
| Source protocol logic | `apps/sense/internal/protocols` | Yes |
| Transport publishing | `apps/sense/internal/app/sense_transport.go` and `shared/mqttx` | Only for new outbound transports |
| Shared UI widgets | `shared/web` | Only for shared UI improvements |

For a new source protocol, start in `apps/sense/internal/protocols`.

### Add a new protocol reader

1. create `myprotocol_reader.go`
2. implement `Reader`
3. register the reader in `registry.go`
4. add a config example
5. add tests

Minimal reader:

```go
type MyProtocolReader struct{}

func (reader MyProtocolReader) Read(ctx context.Context, endpoint SourceEndpoint, metric MetricAddress) (MetricValue, error) {
    value, raw, err := readFromMyProtocol(endpoint, metric)
    if err != nil {
        return MetricValue{}, err
    }
    return MetricValue{
        Name: metric.Name,
        Raw: raw,
        Value: value,
    }, nil
}
```

### Add subscription support

If the protocol supports subscriptions, also implement `Subscriber`.

```go
func (reader MyProtocolReader) Subscribe(ctx context.Context, endpoint SourceEndpoint, metrics []MetricAddress, intervalMS int, handler func(MetricValue)) error {
    return subscribeToMyProtocol(ctx, endpoint, metrics, handler)
}
```

### Register the protocol

```go
func NewRegistry() Registry {
    registry := Registry{
        readers: map[string]Reader{},
        subscribers: map[string]Subscriber{},
    }

    registry.RegisterReader("myprotocol", MyProtocolReader{})
    registry.RegisterSubscriber("myprotocol", MyProtocolReader{})

    return registry
}
```

### Use protocol-specific options

For unusual protocols, use `options` on the source or metric instead of changing the generic config model.

```json
{
  "agentId": "machine-01",
  "type": "myprotocol",
  "origin": "external",
  "host": "s7-adapter-01.local",
  "port": 1234,
  "options": {
    "rack": 0,
    "slot": 1
  },
  "metrics": [
    {
      "name": "state",
      "type": "string",
      "path": "machine.state",
      "options": {
        "encoding": "ascii"
      }
    }
  ]
}
```
