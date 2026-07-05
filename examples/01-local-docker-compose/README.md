# Example 01: Local Docker Compose

This example starts the full OT.io demo stack on one machine.

## Prerequisites

- Docker
- Docker Compose plugin
- Network access to Docker Hub for the first build

## Start

Run from this folder:

```bash
docker compose up --build
```

Or run from the repository root:

```bash
bash scripts/start-local-docker-compose-demo.sh
```

On Windows PowerShell from the repository root:

```powershell
.\scripts\start-local-docker-compose-demo.cmd
```

## Open the UI

```text
http://127.0.0.1:8000
```

## Useful URLs

| Component | URL |
|---|---|
| IoT Lense | http://127.0.0.1:8000 |
| IoT Sense Modbus | http://127.0.0.1:8100 |
| IoT Sense OPC UA | http://127.0.0.1:8101 |
| IoT Dispense Modbus | http://127.0.0.1:8200 |
| IoT Dispense OPC UA | http://127.0.0.1:8201 |
| Presense Modbus 01 | http://127.0.0.1:8301 |
| Presense Modbus 02 | http://127.0.0.1:8302 |
| Presense Modbus 03 | http://127.0.0.1:8303 |
| Presense OPC UA 01 | http://127.0.0.1:4840 |
| Presense OPC UA 02 | http://127.0.0.1:4841 |

## Failure test

Stop one Sense instance:

```bash
docker stop iot-sense-opcua
```

Refresh IoT Lense. The affected Sense node and its source nodes should become unavailable in the connection graph.

Start it again:

```bash
docker start iot-sense-opcua
```
