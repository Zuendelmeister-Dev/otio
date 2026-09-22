# IoT Lense

The [Protocol Lab](protocols.md) adds a searchable protocol catalog, native connection tests and simulator controls. Lense serves it through `/protocols` and the same-origin `/protocol-lab/` proxy; configure `PROTOCOL_LAB_URL` when it runs on another host. Start [Example 03](../examples/03-protocol-lab/README.md) for an integrated demonstration.

IoT Lense is the health, diagnostics and configuration view for OT.io.

Lense does not read machines directly. It observes the messages and component status around the stack, stores telemetry in Postgres and renders the connection graph.

## What Lense shows

- overall health
- broker status
- connected and healthy agents
- recent errors
- connection graph
- Quick Metrics
- Unified Namespace
- logs
- configuration overview

## Data sources

Lense combines three kinds of information:

| Source | Purpose |
|---|---|
| MQTT telemetry and status | Shows what data is flowing. |
| HTTP component probes | Detects stopped Sense, Dispense and Presense containers. |
| Topology configuration | Defines what should exist in the graph. |

## Topology configuration

The default topology file is:

```text
apps/lense/config/topology.json
```

The file is mounted into the Lense container as:

```text
/app/config/topology.json
```

Relevant environment variable:

```text
LENSE_TOPOLOGY_PATH=/app/config/topology.json
```

## Connection graph behavior

| Node | Visual style | Click behavior |
|---|---|---|
| Presense simulation | Dashed grey | Opens simulation options |
| Real external machine | Solid grey | Not clickable |
| Unavailable node | Dashed red | Click behavior depends on node type |
| Sense | Teal | Opens Sense UI or configuration |
| Lense | Orange | Opens local Lense view |
| Dispense | Red | Opens Dispense UI or configuration |
| System components | Yellow | Opens local component details where available |

## Connection graph legend

| Visual element | Meaning |
|---|---|
| Solid grey node | Real external machine or endpoint |
| Dashed grey node | IoT Presense simulation |
| Dashed red node | Unavailable node |
| Grey line | Source connection |
| Teal line | Sense connection |
| Yellow line | System, broker or database connection |
| Red line | Dispense or target connection |
| Dashed red line | Unavailable connection |

## Health checks

Lense actively probes configured modules. This avoids stale MQTT status messages looking healthy after a container has stopped.

Typical probe targets:

| Component | Probe endpoint |
|---|---|
| Sense | `/api/status` |
| Dispense | `/api/status` |
| Presense Modbus | `/api/status` |
| Presense OPC UA | `/health` |

## Configuration workflow

Where available, Lense can open a module configuration page directly.

The workflow is:

1. open Configurations in Lense
2. choose the module group
3. open the module configuration
4. edit JSON
5. validate changes
6. apply proposal
7. verify the graph and health status

## Extension boundary

Lense is intended to stay the central diagnostics, health and configuration surface.

Custom source protocols should be implemented in IoT Sense. Custom outbound integrations should be implemented in IoT Dispense. Lense should only need changes when the central product UI or the shared topology model itself changes.

## PostgreSQL viewer

The integrated viewer under **Agents → System Components → Postgres** lists public tables, previews bounded results and supports a restricted SELECT grammar plus CSV export. It does not require Adminer or a separate client container. See [query limits and usage](ui-workflows.md#postgresql-data-explorer).
