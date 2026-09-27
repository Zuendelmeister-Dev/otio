# Protocol support and Protocol Lab

OT.io separates **simulation** (Presense), **collection** (Sense), and **observation/testing** (Lense). The optional `apps/protocol-lab` service contains native IP drivers and a shared simulator. Lense proxies its UI/API; Sense's `lab-*` readers call its bounded read API, then publish normal OT.io telemetry. Existing `modbus-tcp` and HTTP-demo `opcua` sources remain compatible.

## Support matrix

“Experimental” means a real wire-protocol driver is integrated, but interoperability with physical devices has not been verified here. It does not mean the service substitutes JSON for that protocol. “Gateway” means configure an actual implemented upstream protocol on suitable hardware; OT.io does not decode the original fieldbus itself.

| Requested protocol | Network variant / implementation | Presense and test status |
|---|---|---|
| Modbus TCP / RTU | Native TCP; RTU frames with CRC over TCP tunnel | Both local simulator/client round trips tested. Serial RTU needs a transparent gateway. |
| OPC UA | Native OPC UA Binary over TCP via `lab-opcua-tcp` | Native numeric and boolean round trips tested; None demo mode or certificate profiles with Basic256Sha256 / SignAndEncrypt (anonymous user token). See [certificates](certificates.md). Existing `opcua` remains the older HTTP demo. |
| PROFINET | RT/IRT requires Ethernet controller stack | Gateway; no native IO controller/device simulation. S7 access is not PROFINET IO. |
| PROFIBUS DP/PA | Fieldbus master/gateway | Gateway; no direct TCP driver. |
| EtherNet/IP (CIP) | PLC4Go explicit symbolic tag reads | Experimental `lab-ethernet-ip`; no cyclic I/O adapter simulation. |
| EtherCAT | Raw Ethernet master and suitable interface | Gateway; no native process-I/O implementation. |
| Siemens S7 / ISO-on-TCP / RFC1006 | PLC4Go S7 read over TCP | Experimental `lab-s7`; device requires appropriate access configuration. Optional read-only DB1 REAL simulator on TCP 1102. |
| CAN / CANopen | CAN interface or mapped gateway | Gateway. |
| J1939 | CAN/J1939 interface and PGN/SPN mapping | Gateway. |
| IO-Link | Point-to-point device through an IO-Link master | Gateway using the master's actual upstream API/protocol. |
| HART | Wired HART requires hardware; HART-IP exists | HART-IP is **not implemented**, although IP based. |
| BACnet | BACnet/IP over UDP | Experimental `lab-bacnet-ip` ReadProperty; no MS/TP support. |
| KNX | KNXnet/IP over UDP | Experimental `lab-knxnet-ip` group reads; matching DPT and gateway needed. |
| M-Bus / Wireless M-Bus | Wired bus or radio receiver/gateway | `lab-mbus-tcp`: limited CI72 / DIF02 / VIF5A temperature read through a transparent TCP gateway and matching optional simulator. No serial/radio access. |
| IEC 60870-5-104 | PLC4Go TCP event sampling | Experimental `lab-iec-60870-5-104`; waits for a matching ASDU/IOA event. |
| IEC 61850 | MMS over IP; GOOSE/SV use separate Ethernet mappings | **Not implemented**; no claim of MMS, GOOSE or SV support. |
| DNP3 | TCP and serial variants exist | **Not implemented**; a gateway is currently required. |
| MQTT | MQTT 3.1.1 over TCP | Topic sampling and retained Presense publisher. Real-broker integration test is opt-in / CI. |
| AMQP | AMQP 0-9-1 over TCP | Exchange/routing-key sampling and Presense publisher. AMQP 1.0 is not implemented. Real-broker test is opt-in / CI. |
| AS-Interface (AS-i) | AS-i master/gateway | Gateway. |

## Start and test

Use [Example 03](../examples/03-protocol-lab/README.md) for the complete stack or `go run ./apps/protocol-lab` for the standalone lab UI (simulators off until explicitly started). [Example 04](../examples/04-device-gateways/README.md) covers connection strings and physical-device gateways.

The Lab separates TCP connection testing, Sense deployment preparation, and optional simulation. The selector includes only implemented TCP adapters. UDP and hardware-only entries remain in this reference for compatibility and planning. Simulator activation is explicit through the UI or `LAB_SIMULATORS`. External reads do not activate simulators.

Lense uses `PROTOCOL_LAB_URL` (default `http://protocol-lab:8500`) for its same-origin `/protocol-lab/` proxy. The other demo UIs link to port 8500 on the current host. For a split-host deployment, open the lab's actual URL or configure Lense's proxy target.

## Sense configuration

Add a source to the normal Sense configuration:

```json
{
  "agentId": "plant-s7",
  "type": "lab-s7",
  "host": "192.168.1.10",
  "port": 102,
  "options": {
    "gatewayURL": "http://protocol-lab:8500",
    "connection": "s7://192.168.1.10:102?remote-rack=0&remote-slot=1"
  },
  "metrics": [{"name":"temperature","address":"%DB1:0:REAL","scale":1,"unit":"°C"}]
}
```

The connection is opened from the lab service. `gatewayURL` is optional with the default above; `connection` and a metric `address` or `nodeId` are required. Numeric scaling happens once in Sense. Keep real credentials out of committed configuration files.

All lab sources currently use Sense polling. For event-oriented protocols, each request samples one event during a short-lived subscription. This is useful for diagnostics, but can miss events between requests and is not a lossless event historian. AMQP creates its own exclusive auto-delete queue bound to an existing exchange, so a test does not drain another consumer's queue.

## API and limits

- `GET /api/catalog`: implemented operations and gateway requirements.
- `POST /api/read`: `{ "protocol": "opcua-tcp", "connection": "opc.tcp://localhost:4842", "address": "ns=1;s=Temperature" }` → value, protocol, latency and timestamp.
- `GET /api/simulator`, `PUT /api/simulator`: simulator values and publisher status; PUT accepts temperature, running and mode.
- `GET /health`: lab HTTP service health.
- `GET /api/status`: service and simulator status used by Lense's component monitor.

Reads have an eight-second context timeout, eight concurrent slots and a 16 KiB HTTP request limit. Network/device failures return errors rather than fabricated values. MQTT/AMQP JSON samples are limited to 1 MiB after receipt. This is a trusted-lab diagnostic service, not an authenticated production gateway. There is no write API for external devices.

## Dependencies and verification

PLC4Go is pinned to `v0.0.0-20260911155029-fd6223dd098d`, which requires Go 1.27. This is an upstream development revision; its API/driver behavior is covered by local address-validation and native Modbus round-trip tests. OPC UA uses gopcua 0.9.1. The workspace selects Go 1.27.1; the lab container uses the same version.

Tests cover catalog/URL validation, every experimental driver's default address grammar, HTTP failure handling, simulator input validation, RTU CRC/Modbus exception handling, real TCP/RTU/OPC-UA round trips, Sense gateway scaling/errors, and Lense proxy routing. Set `LAB_TEST_MQTT_URL` and `LAB_TEST_AMQP_URL` to exercise actual brokers. Those tests skip explicitly when endpoints are absent. Docker image builds and physical PLC/fieldbus tests are separate checks.

## Primary references

- [Apache PLC4X protocol support](https://plc4x.apache.org/plc4x/pre-release/users/protocols/index.html) and [PLC4Go API](https://plc4x.apache.org/plc4x/pre-release/users/getting-started/plc4go.html).
- [gopcua native OPC UA implementation](https://github.com/gopcua/opcua).
- [FieldComm Group HART-IP](https://www.fieldcommgroup.org/technologies/HART-IP).
- [IEC 61850-8-1 mappings](https://webstore.iec.ch/en/publication/66585).
- [IO-Link technology](https://io-link.com/) and [DNP3 overview](https://www.dnp.org/About/Overview-of-DNP3-Protocol).

The limited M-Bus fixture follows the [documented FT1.2 frame format](https://m-bus.com/documentation-wired/05-data-link-layer) and [variable-data layout](https://m-bus.com/documentation-wired/06-application-layer); unsupported encodings, units and corrupt checksums fail explicitly. Validate against the specific gateway and meter before using it beyond the demo.
