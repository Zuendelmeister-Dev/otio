# Docker Compose Example

The root `docker-compose.yml` is the recommended first way to run OT.io.

It starts:

- Mosquitto as the input MQTT broker
- Mosquitto as the target MQTT broker for Dispense
- Postgres as historian database
- A small read-only PostgreSQL data explorer inside Lense at <http://localhost:8000/#agents/postgres>
- three Modbus Presense simulators
- two OPC UA style Presense simulators
- Sense Modbus in polling mode
- Sense OPC UA in subscription mode
- Lense as central health and configuration view
- Dispense Modbus and Dispense OPC UA as forwarding modules

## Start

```bash
docker compose up --build
```

## First checks

1. open IoT Lense at http://127.0.0.1:8000
2. check the connection graph
3. open IoT Sense Modbus at http://127.0.0.1:8100
4. open IoT Sense OPC UA at http://127.0.0.1:8101
5. open IoT Dispense OPC UA at http://127.0.0.1:8201

## Test an outage

```bash
docker stop iot-sense-opcua
```

Lense should mark the OPC UA Sense path and its source nodes as unavailable.

Start it again:

```bash
docker start iot-sense-opcua
```

## Switch OPC UA Sense back to polling

Edit `apps/sense/config-opcua/config.json` and change each OPC UA source:

```json
"readMode": "poll"
```

Then restart the Sense OPC UA container:

```bash
docker restart iot-sense-opcua
```

## Connect a real endpoint

1. Edit the Sense configuration.
2. Replace the Presense `host` and `port` with the real endpoint.
3. Update `apps/lense/config/topology.json` so the graph knows the new source.
4. Restart the affected Sense instance.

Real external endpoints should be visible in Lense, but they should not be clickable because they are not managed OT.io modules.
