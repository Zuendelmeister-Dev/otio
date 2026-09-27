# Example 03 — Protocol Lab

A standalone demo stack with the new Presense simulator, native protocol adapters, Sense collection, MQTT, RabbitMQ, Postgres and Lense. Stop Example 01 first: this example uses the same UI and broker ports.

## Start

From this directory:

```sh
docker compose up --build -d
docker compose logs -f protocol-lab sense
```

Docker must be running. The first build downloads a Go 1.27 toolchain and the PLC4Go drivers, so it takes longer than the smaller legacy demos.

Open [Protocol Lab](http://localhost:8500/static/), [Lense](http://localhost:8000) or [Sense](http://localhost:8100). Lense's **Protocol Lab** link proxies the lab through Lense's own origin. RabbitMQ management is at [localhost:15672](http://localhost:15672), with demo login `lab` / `lab`.

## Try it

1. Select **Modbus TCP**, keep its default address, and choose **Read a value**. The raw result starts at `2350` (23.50 °C).
2. Open **Simulation**, change **Temperature** in the Presense playground and choose **Update simulator**. Read again to verify the new wire value.
3. Select **OPC UA Binary**. `ns=1;s=Temperature` returns °C directly; `ns=1;s=Running` returns a boolean.
4. Try **Modbus RTU over TCP**. This sends actual RTU frames with CRC through a TCP tunnel; it does not access a serial port.
5. For MQTT use `tcp://mqtt:1883`, topic `lab/temperature`. For AMQP use `amqp://lab:lab@rabbitmq:5672/`, address `amq.topic/lab.temperature`. Broker names resolve from the lab container. Publishers retry after startup or connection errors.
6. Open Lense's metrics to see Sense collecting the five demo sources. Sense scales Modbus temperature by 0.01. String/boolean values remain non-numeric telemetry.
7. Use **Test connection** for an external TCP device. **Deploy Sense** prepares a collector deployment. UDP and hardware-only protocols are omitted from this focused selector; see the protocol matrix.

The simulator supports constant values or a ±2 °C sine wave and a boolean running state. The HTTP API changes only the simulator; the external device test API exposes no write operation.

## Endpoints

| Interface | Host port | Details |
|---|---:|---|
| Protocol Lab UI/API | 8500 | `/static/`, `/api/catalog`, `/api/read`, `/api/simulator` |
| Modbus TCP | 1502 | unit 1, zero-based register 0 = °C ×100, register 1 = running |
| Modbus RTU over TCP | 1503 | same register map; RTU framing + CRC |
| OPC UA Binary | 4842 | anonymous / None; `ns=1;s=Temperature`, `ns=1;s=Running` |
| S7 / RFC1006 | 1102 | opt-in; DB1 bytes 0–3 as REAL; controller-type=S7_1200 |
| M-Bus over TCP | 1504 | opt-in; primary 1, CI72 / DIF02 / VIF5A temperature subset |
| MQTT | 1883 | retained JSON temperature at `lab/temperature` |
| AMQP 0-9-1 | 5672 | exchange `amq.topic`, routing key `lab.temperature` |

All published ports bind to loopback. These are demo credentials and anonymous lab endpoints. For another machine, configure reachability and access controls deliberately.

## Stop and reset

```sh
docker compose down
```

Postgres history remains in the named volume. `docker compose down -v` also deletes that demo history. Simulator values reset on restart. Sense configuration edits persist in the mounted `config/` directory.

## Without Docker

From the repository root, with Go 1.27.1:

```sh
go run ./apps/protocol-lab
```

This starts only the lab UI/API. Open **Simulation** to start a selected listener, or set `LAB_SIMULATORS` to a comma-separated list: `modbus-tcp,modbus-rtu-tcp,opcua-tcp,s7,mbus-tcp,mqtt,amqp`. MQTT and AMQP additionally need `LAB_MQTT_URL` and/or `LAB_AMQP_URL`. The Compose example explicitly starts its five existing demo sources; S7 and M-Bus remain opt-in. It does not start Lense, Sense, Postgres or broker services.

For physical devices see [Example 04](../04-device-gateways/README.md). The [protocol matrix](../../docs/protocols.md) distinguishes working demo interfaces, experimental device adapters and missing native support.
