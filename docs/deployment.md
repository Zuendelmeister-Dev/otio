# OT.io Deployment

The first supported deployment model is Docker Compose. Individual Docker runs are also possible and are useful for understanding how the modules fit together.

## Docker Compose deployment

```bash
docker compose up --build
```

Stop the stack:

```bash
docker compose down --remove-orphans
```

Rebuild cleanly:

```bash
docker compose build --no-cache
docker compose up
```

## Module image builds

From the repository root:

```bash
docker build -t otio/presense-modbus:beta apps/presense-modbus
docker build -t otio/presense-opcua:beta -f apps/presense-opcua/Dockerfile .
docker build -t otio/sense:beta -f apps/sense/Dockerfile .
docker build -t otio/lense:beta -f apps/lense/Dockerfile .
docker build -t otio/dispense:beta -f apps/dispense/Dockerfile .
```

## Minimal manual rollout order

1. create Docker network
2. start MQTT broker
3. start Postgres
4. start Presense or real endpoints
5. start Sense
6. start Lense
7. start Dispense and target systems

```bash
docker network create otio-net
```

Start input broker:

```bash
docker run -d --name mqtt --network otio-net -p 1883:1883 eclipse-mosquitto:2
```

Start target broker:

```bash
docker run -d --name dispense-broker --network otio-net -p 1884:1883 eclipse-mosquitto:2
```

Start Postgres:

```bash
docker run -d --name postgres --network otio-net   -e POSTGRES_USER=otio   -e POSTGRES_PASSWORD=otio   -e POSTGRES_DB=otio   -p 5432:5432 postgres:16
```

Start Lense:

```bash
docker run -d --name lense --network otio-net -p 8000:8000   -e POSTGRES_HOST=postgres   -e POSTGRES_PORT=5432   -e POSTGRES_DB=otio   -e POSTGRES_USER=otio   -e POSTGRES_PASSWORD=otio   -e MQTT_HOST=mqtt   -e MQTT_PORT=1883   -e LENSE_TOPOLOGY_PATH=/app/config/topology.json   -v "$PWD/apps/lense/config:/app/config:ro"   otio/lense:beta
```

Start Sense OPC UA:

```bash
docker run -d --name sense-opcua --network otio-net -p 8101:8100   -v "$PWD/apps/sense/config-opcua:/app/config:ro"   otio/sense:beta
```

Start Dispense OPC UA:

```bash
docker run -d --name dispense-opcua --network otio-net -p 8201:8201   -e DISPENSE_INSTANCE_ID=iot-dispense-opcua-01   -e DISPENSE_SOURCE_FILTER=opcua   -e DISPENSE_SOURCE_PREFIX=iot-lense   -e DISPENSE_TARGET_PREFIX=dispense/opcua   -e DISPENSE_INPUT_BROKER_HOST=mqtt   -e DISPENSE_INPUT_BROKER_PORT=1883   -e DISPENSE_OUTPUT_BROKER_HOST=dispense-broker   -e DISPENSE_OUTPUT_BROKER_PORT=1883   -e DISPENSE_UI_PORT=8201   otio/dispense:beta
```

## Production gap list

Before production usage, add:

- authentication
- authorization
- TLS certificates
- encrypted broker traffic
- secret handling
- backup and restore
- operational monitoring
- alerting
- Kubernetes manifests or Helm charts
