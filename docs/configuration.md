# OT.io Configuration Guide

OT.io uses JSON configuration and environment variables. The local Docker Compose test stack mounts the configuration files into the containers.

## Sense configuration

Sense configuration defines:

- MQTT broker connection.
- Poll interval.
- Health timeout.
- Sources to read.
- Metrics per source.

Example Modbus source:

```json
{
  "agentId": "press-line-01",
  "type": "modbus-tcp",
  "host": "192.168.10.42",
  "port": 502,
  "unitId": 1,
  "origin": "external",
  "metrics": [
    {
      "name": "temperature",
      "register": 0,
      "scale": 0.1,
      "unit": "°C",
      "type": "gauge"
    }
  ]
}
```

Example Presense simulation source:

```json
{
  "agentId": "modbus-machine-01",
  "type": "modbus-tcp",
  "host": "presense-modbus-01",
  "port": 5020,
  "unitId": 1,
  "origin": "presense",
  "metrics": []
}
```

The `origin` field is optional in the current demo, but it is recommended. Use `presense` for simulators and `external` for real machines.

## Lense topology configuration

Lense reads topology metadata from `apps/lense/config/topology.json` in the local demo. The topology tells Lense which OT.io modules exist, which local or remote UI URLs they have and how the graph should be assembled.

Lense does not own real machine configuration directly. It shows external endpoints as graph nodes and makes Sense configuration reachable where needed.

## Dispense configuration

Dispense uses environment variables in the current demo:

| Variable | Meaning |
|---|---|
| `DISPENSE_INSTANCE_ID` | Logical Dispense instance id. |
| `DISPENSE_SOURCE_FILTER` | Source type filter such as `modbus-tcp` or `opcua`. |
| `DISPENSE_SOURCE_PREFIX` | Input MQTT topic prefix. |
| `DISPENSE_TARGET_PREFIX` | Output MQTT topic prefix. |
| `DISPENSE_INPUT_BROKER_HOST` | Input broker host. |
| `DISPENSE_INPUT_BROKER_PORT` | Input broker port. |
| `DISPENSE_OUTPUT_BROKER_HOST` | Target broker host. |
| `DISPENSE_OUTPUT_BROKER_PORT` | Target broker port. |
| `DISPENSE_UI_PORT` | HTTP UI port. |

## Presense configuration

Presense configuration is currently environment-variable based.

Common variables:

| Variable | Meaning |
|---|---|
| `DEVICE_ID` | Simulated endpoint id. |
| `GENERATOR_MODE` | Value generation mode, for example `sine`, `sawtooth`, `triangle` or `square`. |

Presense is only a simulator. Production deployments can remove Presense completely and point Sense to real endpoints.
