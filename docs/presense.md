# IoT Presense

IoT Presense is the simulator part of OT.io. I use it to create predictable source endpoints while developing and testing Sense, Lense and Dispense.

Presense is optional. In a real deployment, Sense can point directly to real machines or external endpoints.

## Supported simulator types

| Module | Simulated protocol | Current behavior |
|---|---|---|
| `apps/presense-modbus` | Modbus TCP | Exposes generated values through holding registers. |
| `apps/presense-opcua` | OPC UA style endpoint | Exposes generated values through `/read`, `/nodes` and `/subscribe`. |

## Modbus Presense

The Modbus simulator exposes generated values on holding registers.

Default registers used by the demo:

| Register | Metric | Unit |
|---:|---|---|
| `0` | `temperature` | `°C` |
| `1` | `humidity` | `%` |
| `2` | `pressure` | `bar` |
| `3` | `vibration` | `mm/s` |

Example Docker Compose service:

```yaml
presense-modbus-01:
  build: ./apps/presense-modbus
  container_name: presense-modbus-01
  restart: unless-stopped
  environment:
    DEVICE_ID: modbus-machine-01
    MODBUS_PORT: 5020
    TEMP_BASE: 22.0
    HUMIDITY_BASE: 48.0
    PRESSURE_BASE: 1.20
    VIBRATION_BASE: 0.40
    PRESENSE_UI_PORT: 8300
    GENERATOR_MODE: sine
  ports:
    - "5020:5020"
    - "8301:8300"
```

## OPC UA style Presense

The OPC UA style simulator exposes node values through a small HTTP demo API. This is not yet a hardened OPC UA server with certificates. It is the beta simulation layer used to model OPC UA reads and subscriptions.

Default nodes:

| Node id | Metric | Type | Unit |
|---|---|---|---|
| `ns=2;s=Machine.Temperature` | `temperature` | numeric | `°C` |
| `ns=2;s=Machine.Speed` | `speed` | numeric | `rpm` |
| `ns=2;s=Machine.Current` | `current` | numeric | `A` |
| `ns=2;s=Machine.Load` | `load` | numeric | `%` |
| `ns=2;s=Machine.State` | `state` | string | none |

Read one node:

```text
GET /read?nodeId=ns=2;s=Machine.Temperature
```

List all demo nodes:

```text
GET /nodes
```

Subscribe to selected nodes:

```text
GET /subscribe?nodeIds=ns=2;s=Machine.Temperature,ns=2;s=Machine.State&intervalMs=1000
```

The subscription endpoint returns Server-Sent Events. Each event contains a timestamp and an `items` array with node values.

Example event payload:

```json
{
  "timestamp": "2026-05-22T18:40:00Z",
  "items": [
    {
      "nodeId": "ns=2;s=Machine.Temperature",
      "value": 31.4,
      "quality": "good",
      "unit": "°C"
    },
    {
      "nodeId": "ns=2;s=Machine.State",
      "value": "running",
      "quality": "good"
    }
  ]
}
```

Example Docker Compose service:

```yaml
presense-opcua-01:
  build:
    context: .
    dockerfile: apps/presense-opcua/Dockerfile
  container_name: presense-opcua-01
  restart: unless-stopped
  environment:
    DEVICE_ID: opcua-machine-01
    OPCUA_HTTP_PORT: 4840
    TEMP_BASE: 31.0
    SPEED_BASE: 1420.0
    CURRENT_BASE: 8.2
    GENERATOR_MODE: sine
  ports:
    - "4840:4840"
```

## Generator modes

| Mode | Result |
|---|---|
| `sine` | Smooth wave-like numeric value. |
| `sawtooth` | Repeating rising numeric value. |
| `triangle` | Rising and falling numeric value. |
| `square` | Alternating numeric levels. |
| `random-int` | Random integer-like value. |
| `random-string` | String-like state simulation. |
| `array` | Array-like sample behavior in the simulator. |

## How Presense appears in Lense

Presense nodes are shown with a dashed grey border in the connection graph. Real external machines use a solid grey border and are not clickable.

## Writing a custom Presense extension

A Presense extension is useful when I want to simulate a protocol or a machine behavior before a real endpoint exists. It should look like a source endpoint to Sense, but it is still part of the OT.io lab world.

### Recommended folder layout

```text
apps/
└── presense-myprotocol/
    ├── Dockerfile
    ├── go.mod
    ├── main.go
    ├── config.go
    ├── generator.go
    ├── server.go
    └── server_test.go
```

### Minimal responsibilities

A custom Presense module should:

1. load configuration from environment variables
2. expose one protocol endpoint or one HTTP-compatible simulation endpoint
3. generate deterministic demo values
4. expose health/status endpoints for Lense
5. return enough metadata so Sense can identify the endpoint
6. include tests for configuration parsing and generated values

### Required HTTP endpoints

Even if the simulated protocol is not HTTP-based, the module should expose a small HTTP surface for diagnostics.

| Endpoint | Purpose |
|---|---|
| `/api/status` or `/health` | Allows Lense to detect whether the simulator is alive. |
| `/api/config` | Optional, shows the currently active simulator configuration. |
| `/nodes` | Recommended for node-like protocols such as OPC UA. |
| `/read` | Recommended for simple read tests. |
| `/subscribe` | Recommended for stream/subscription tests. |

### Basic environment variables

```env
DEVICE_ID=my-simulated-machine-01
PROTOCOL=myprotocol
LISTEN_ADDR=0.0.0.0:8500
GENERATOR_MODE=sine
BASE_VALUE=20
AMPLITUDE=5
STEP=0.2
NOISE=0.1
```

### Example Docker Compose service

```yaml
presense-myprotocol-01:
  build:
    context: .
    dockerfile: apps/presense-myprotocol/Dockerfile
  container_name: presense-myprotocol-01
  environment:
    DEVICE_ID: my-simulated-machine-01
    PROTOCOL: myprotocol
    LISTEN_ADDR: 0.0.0.0:8500
    GENERATOR_MODE: sine
    BASE_VALUE: "20"
    AMPLITUDE: "5"
    STEP: "0.2"
  ports:
    - "8500:8500"
```

### How Sense should consume it

The matching Sense source should point to the Presense service name inside the Docker network.

```json
{
  "agentId": "my-simulated-machine-01",
  "displayName": "My Simulated Machine 01",
  "type": "myprotocol",
  "origin": "presense",
  "host": "presense-myprotocol-01",
  "port": 8500,
  "metrics": [
    {
      "name": "temperature",
      "type": "gauge",
      "unit": "°C",
      "path": "temperature"
    }
  ]
}
```

`origin` should be `presense` so Lense can render it as a dashed simulation node and allow navigation to the simulator UI.

### Testing checklist

Before adding the module to the main demo stack:

- test configuration parsing
- test every generator mode
- test health endpoint responses
- test a sample read
- test subscription/stream behavior if implemented
- test the matching Sense source configuration
