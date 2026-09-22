# OT.io Configuration and Rollout Guide

This guide explains how I configure and run OT.io. It starts with the Docker Compose demo because that is the easiest way to play with the project, then shows how the modules can be started individually.

The current beta uses three configuration mechanisms:

1. environment variables for simple runtime settings
2. JSON configuration files for Sense and Lense
3. JSON stored inside `DISPENSE_CONFIG_JSON` or individual environment variables for Dispense

## 1. Fastest start with Docker Compose

From the repository root:

```bash
docker compose up --build
```

Open Lense:

```text
http://127.0.0.1:8000
```

Stop everything:

```bash
docker compose down --remove-orphans
```

Rebuild without cache:

```bash
docker compose build --no-cache
docker compose up
```

## 2. How the demo stack is wired

The Compose stack starts:

- `mqtt` as the input broker on port `1883`
- `dispense-broker` as the target broker on port `1884`
- `postgres` as historian storage
- three Modbus Presense simulators
- two OPC UA style Presense simulators
- one Sense Modbus collector
- one Sense OPC UA collector with subscription mode
- one Lense instance
- two Dispense instances

## 3. IoT Presense configuration

Presense is a simulator. I use it to test Sense without real machines.

Production deployments can remove Presense and point Sense to real endpoints.

### Modbus Presense service

```yaml
presense-modbus-01:
  build:
    context: .
    dockerfile: apps/presense-modbus/Dockerfile
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

### OPC UA style Presense service

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

### Presense environment variables

| Variable | Used by | Meaning |
|---|---|---|
| `DEVICE_ID` | Modbus, OPC UA | Simulated endpoint id. |
| `GENERATOR_MODE` | Modbus, OPC UA | `sine`, `sawtooth`, `triangle`, `square`, `random-int`, `random-string` or `array`. |
| `MODBUS_PORT` | Modbus | Modbus TCP listen port inside the container. |
| `PRESENSE_UI_PORT` | Modbus | Presense UI/API port inside the container. |
| `OPCUA_HTTP_PORT` | OPC UA | OPC UA style demo API port. |
| `TEMP_BASE` | Modbus, OPC UA | Base temperature value. |
| `HUMIDITY_BASE` | Modbus | Base humidity value. |
| `PRESSURE_BASE` | Modbus | Base pressure value. |
| `VIBRATION_BASE` | Modbus | Base vibration value. |
| `SPEED_BASE` | OPC UA | Base speed value. |
| `CURRENT_BASE` | OPC UA | Base current value. |

### OPC UA style subscription endpoint

Presense OPC UA exposes two read patterns:

```text
GET /read?nodeId=ns=2;s=Machine.Temperature
GET /subscribe?nodeIds=ns=2;s=Machine.Temperature,ns=2;s=Machine.State&intervalMs=1000
```

`/subscribe` streams Server-Sent Events with JSON payloads. This models an OPC UA subscription for the beta demo. Certificates and encrypted OPC UA sessions are planned later.

## 4. IoT Sense configuration

Sense reads source endpoints and publishes metric, status and error messages to MQTT.

The Docker Compose demo mounts these directories:

| Instance | Mounted directory |
|---|---|
| Sense Modbus | `apps/sense/config-modbus` |
| Sense OPC UA | `apps/sense/config-opcua` |

The file inside the container is:

```text
/app/config/config.json
```

### Supported Sense source types

| `source.type` | Read mode | What it does |
|---|---|---|
| `modbus-tcp` | `poll` | Reads Modbus holding registers. |
| `opcua` | `poll` | Calls the OPC UA style `/read` endpoint. |
| `opcua` | `subscription` | Consumes the OPC UA style `/subscribe` stream. |

### Sense configuration fields

| Field | Meaning |
|---|---|
| `broker.host` | MQTT broker host. |
| `broker.port` | MQTT broker port. |
| `broker.clientId` | MQTT client id used by this Sense instance. |
| `broker.topicPrefix` | Prefix used for telemetry, status and error topics. |
| `pollIntervalMs` | Polling interval for poll mode. Also used as fallback for subscription interval. |
| `healthTimeoutSeconds` | Timeout used by UIs and health calculations. |
| `sources[].agentId` | Source id shown in Lense and used in MQTT topics. |
| `sources[].type` | `modbus-tcp` or `opcua`. |
| `sources[].host` | Source host inside the Docker network or real network. |
| `sources[].port` | Source port. |
| `sources[].unitId` | Modbus unit id. Required for Modbus. |
| `sources[].readMode` | `poll` or `subscription`. Default is `poll`. Subscription is currently OPC UA only. |
| `sources[].subscriptionIntervalMs` | OPC UA subscription interval. Minimum is 250 ms. |
| `sources[].metrics[].name` | Metric name. |
| `sources[].metrics[].register` | Modbus register address. |
| `sources[].metrics[].nodeId` | OPC UA node id. |
| `sources[].metrics[].scale` | Numeric scale. Must not be 0. |
| `sources[].metrics[].unit` | Display unit. |
| `sources[].metrics[].type` | `gauge`, `string` or another descriptive metric type. Numeric values are charted, non-numeric values are shown as tables. |

### Complete Sense Modbus example

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
        },
        {
          "name": "pressure",
          "register": 2,
          "scale": 0.01,
          "unit": "bar",
          "type": "gauge"
        },
        {
          "name": "vibration",
          "register": 3,
          "scale": 0.01,
          "unit": "mm/s",
          "type": "gauge"
        }
      ]
    }
  ]
}
```

### Complete Sense OPC UA subscription example

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
          "name": "speed",
          "nodeId": "ns=2;s=Machine.Speed",
          "scale": 1,
          "unit": "rpm",
          "type": "gauge"
        },
        {
          "name": "current",
          "nodeId": "ns=2;s=Machine.Current",
          "scale": 1,
          "unit": "A",
          "type": "gauge"
        },
        {
          "name": "load",
          "nodeId": "ns=2;s=Machine.Load",
          "scale": 1,
          "unit": "%",
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

### Connecting a real machine

To connect a real machine, replace the Presense host with the machine address.

Example:

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

Also update `apps/lense/config/topology.json` so Lense knows where the source belongs in the connection graph.

## 5. IoT Lense topology configuration

Lense uses `apps/lense/config/topology.json` to know the expected demo topology. This is separate from the telemetry stream. Telemetry tells Lense what happened. Topology tells Lense what should exist.

Minimal example:

```json
{
  "senses": [
    {
      "id": "iot-sense-modbus-01",
      "label": "IoT Sense Modbus",
      "url": "http://127.0.0.1:8100",
      "sourceType": "modbus-tcp",
      "internalUrl": "http://sense-modbus:8100"
    }
  ],
  "agents": [
    {
      "agentId": "modbus-machine-01",
      "sourceType": "modbus-tcp",
      "sourceHost": "presense-modbus-01",
      "senseId": "iot-sense-modbus-01"
    }
  ],
  "presenses": [
    {
      "agentId": "modbus-machine-01",
      "sourceType": "modbus-tcp",
      "sourceHost": "presense-modbus-01",
      "senseId": "iot-sense-modbus-01",
      "url": "http://127.0.0.1:8301",
      "internalUrl": "http://presense-modbus-01:8300"
    }
  ],
  "dispenses": [
    {
      "id": "iot-dispense-modbus-01",
      "label": "IoT Dispense Modbus",
      "url": "http://127.0.0.1:8200",
      "sourceType": "modbus-tcp",
      "target": "dispense-target-mqtt",
      "internalUrl": "http://dispense-modbus:8200"
    }
  ],
  "targets": [
    {
      "id": "dispense-target-mqtt",
      "label": "Dispense Target MQTT",
      "host": "dispense-broker",
      "port": 1883,
      "externalPort": 1884
    }
  ]
}
```

## 6. IoT Dispense configuration

Dispense currently reads configuration from environment variables. You can either set individual variables or pass a full JSON object through `DISPENSE_CONFIG_JSON`.

### Individual environment variables

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
| `DISPENSE_OUTPUT_BROKER_HOST` | Output broker host. |
| `DISPENSE_OUTPUT_BROKER_PORT` | Output broker port. |
| `DISPENSE_OUTPUT_CLIENT_ID` | Output MQTT client id. |
| `DISPENSE_UI_PORT` | Dispense UI/API port. |

### Complete Dispense JSON example

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

Run with JSON configuration:

```bash
docker run --rm --network otio-net -p 8201:8201   -e DISPENSE_UI_PORT=8201   -e DISPENSE_CONFIG_JSON='{"instanceId":"iot-dispense-opcua-01","sourceFilter":"opcua","sourcePrefix":"iot-lense","targetPrefix":"dispense/opcua","bufferLimit":1500,"inputBroker":{"host":"mqtt","port":1883,"clientId":"iot-dispense-opcua-01-in"},"outputBroker":{"host":"dispense-broker","port":1883,"clientId":"iot-dispense-opcua-01-out"}}'   otio/dispense:beta
```

## 7. Running modules without Docker Compose

Create a network:

```bash
docker network create otio-net
```

Start brokers:

```bash
docker run -d --name mqtt --network otio-net -p 1883:1883 eclipse-mosquitto:2
docker run -d --name dispense-broker --network otio-net -p 1884:1883 eclipse-mosquitto:2
```

Start Postgres:

```bash
docker run -d --name postgres --network otio-net   -e POSTGRES_USER=otio   -e POSTGRES_PASSWORD=otio   -e POSTGRES_DB=otio   -p 5432:5432 postgres:16
```

Build module images:

```bash
docker build -t otio/presense-modbus:beta -f apps/presense-modbus/Dockerfile .
docker build -t otio/presense-opcua:beta -f apps/presense-opcua/Dockerfile .
docker build -t otio/sense:beta -f apps/sense/Dockerfile .
docker build -t otio/lense:beta -f apps/lense/Dockerfile .
docker build -t otio/dispense:beta -f apps/dispense/Dockerfile .
```

Run a Presense OPC UA simulator:

```bash
docker run -d --name presense-opcua-01 --network otio-net -p 4840:4840   -e DEVICE_ID=opcua-machine-01   -e OPCUA_HTTP_PORT=4840   -e GENERATOR_MODE=sine   otio/presense-opcua:beta
```

Run Sense with a mounted config:

```bash
docker run -d --name sense-opcua --network otio-net -p 8101:8100   -v "$PWD/apps/sense/config-opcua:/app/config:ro"   otio/sense:beta
```

Run Lense with topology:

```bash
docker run -d --name lense --network otio-net -p 8000:8000   -e POSTGRES_HOST=postgres   -e POSTGRES_PORT=5432   -e POSTGRES_DB=otio   -e POSTGRES_USER=otio   -e POSTGRES_PASSWORD=otio   -e MQTT_BROKER=mqtt   -e MQTT_PORT=1883   -e LENSE_TOPOLOGY_PATH=/app/config/topology.json   -v "$PWD/apps/lense/config:/app/config:ro"   otio/lense:beta
```

Run Dispense:

```bash
docker run -d --name dispense-opcua --network otio-net -p 8201:8201   -e DISPENSE_INSTANCE_ID=iot-dispense-opcua-01   -e DISPENSE_SOURCE_FILTER=opcua   -e DISPENSE_SOURCE_PREFIX=iot-lense   -e DISPENSE_TARGET_PREFIX=dispense/opcua   -e DISPENSE_INPUT_BROKER_HOST=mqtt   -e DISPENSE_INPUT_BROKER_PORT=1883   -e DISPENSE_OUTPUT_BROKER_HOST=dispense-broker   -e DISPENSE_OUTPUT_BROKER_PORT=1883   -e DISPENSE_UI_PORT=8201   otio/dispense:beta
```

## 8. Runtime configuration through the UI

Where implemented, the workflow is:

1. open the component UI or the Lense configuration overview
2. open the configuration page
3. edit the JSON
4. validate changes
5. review the diff
6. apply the proposal
7. check Lense health and the connection graph

## 9. Production note

The current stack is a beta demo and development setup. For production usage, add at least:

- authentication
- authorization
- TLS and certificates
- encrypted broker communication
- broker credentials
- secret management
- audit logging
- hardened network policies
- backup and restore
- operational monitoring
- alerting

In the HA example, Sense sets `OTIO_CONFIG_READ_ONLY=true`: apply/rollback through the API is disabled, and the response directs users to Helm. Both replicas mount one shared ConfigMap read-only; its checksum triggers rollout. Broker credentials come from `OTIO_MQTT_USERNAME`/`OTIO_MQTT_PASSWORD` Secrets. Lense accepts `POSTGRES_DSN` for a complete PostgreSQL URI (including TLS parameters); it takes precedence over individual connection fields.
