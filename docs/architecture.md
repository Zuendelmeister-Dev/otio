# OT.io Architecture

OT.io is a lightweight connectivity and diagnostics platform. It connects machine data to target systems and helps me find broken links quickly.

The documentation uses Mermaid diagrams directly in Markdown so GitHub can render the diagrams without a separate build step.

## Introduction and goals

OT.io provides a small modular stack for:

- simulating protocol endpoints for test scenarios
- collecting machine and simulator data
- transporting telemetry and health data through MQTT
- observing health and message flow
- configuring modules quickly
- forwarding selected data to target systems

The main design goals are:

- simple integration
- simple extension
- fast fault localization
- visible data flow
- reusable module and UI building blocks

## Context View

```mermaid
flowchart LR
    Operator["Operator"]
    Machine["External machine endpoint<br/>not managed by OT.io"]

    subgraph Stack["OT.io demo stack"]
        Presense["IoT Presense<br/>simulation only"]
        Sense["IoT Sense<br/>collection"]
        Broker["MQTT Broker<br/>transport"]
        Lense["IoT Lense<br/>health and quick configuration"]
        Postgres[("Postgres<br/>historian")]
        Dispense["IoT Dispense<br/>distribution"]
        Target["Target systems"]
    end

    Machine -->|protocol read| Sense
    Presense -.->|simulated protocol endpoint| Sense
    Sense -->|telemetry and status| Broker
    Broker -->|subscribed messages| Lense
    Lense -->|store and query| Postgres
    Broker -->|subscribed messages| Dispense
    Dispense -->|forward selected data| Target
    Operator -->|observe and configure| Lense

    classDef presense fill:#1b1f27,stroke:#8a9099,stroke-dasharray:5 5,color:#e7e9ee;
    classDef sense fill:#1b1f27,stroke:#14b8a6,color:#e7e9ee;
    classDef lense fill:#1b1f27,stroke:#ff8a1d,color:#e7e9ee;
    classDef dispense fill:#1b1f27,stroke:#ef4444,color:#e7e9ee;
    classDef system fill:#1b1f27,stroke:#ffd21f,color:#e7e9ee;
    classDef external fill:#1b1f27,stroke:#8a9099,color:#e7e9ee;

    class Presense presense;
    class Sense sense;
    class Lense lense;
    class Dispense dispense;
    class Broker,Postgres,Target system;
    class Machine,Operator external;
```

## Container View

```mermaid
flowchart LR
    subgraph DemoStack["OT.io Docker Compose demo stack"]
        PM["Presense Modbus<br/>simulated Modbus TCP endpoint"]
        PO["Presense OPC UA<br/>simulated OPC UA endpoint"]
        SM["Sense Modbus<br/>polls Modbus sources"]
        SO["Sense OPC UA<br/>subscribes to OPC UA sources"]
        MQTT["MQTT Broker<br/>Mosquitto input broker"]
        LENSE["IoT Lense<br/>health, quick metrics, configuration overview"]
        PG[("Postgres<br/>historian database")]
        DM["Dispense Modbus<br/>forwards Modbus-origin messages"]
        DO["Dispense OPC UA<br/>forwards OPC-UA-origin messages"]
        TARGET["Target MQTT Broker<br/>outbound demo target"]
    end

    PM -.->|protocol read| SM
    PO -.->|subscription stream| SO
    SM -->|MQTT publish| MQTT
    SO -->|MQTT publish| MQTT
    MQTT -->|MQTT subscribe| LENSE
    LENSE -->|SQL writes and reads| PG
    MQTT -->|MQTT subscribe| DM
    MQTT -->|MQTT subscribe| DO
    DM -->|MQTT forward| TARGET
    DO -->|MQTT forward| TARGET

    classDef presense fill:#1b1f27,stroke:#8a9099,stroke-dasharray:5 5,color:#e7e9ee;
    classDef sense fill:#1b1f27,stroke:#14b8a6,color:#e7e9ee;
    classDef lense fill:#1b1f27,stroke:#ff8a1d,color:#e7e9ee;
    classDef dispense fill:#1b1f27,stroke:#ef4444,color:#e7e9ee;
    classDef system fill:#1b1f27,stroke:#ffd21f,color:#e7e9ee;

    class PM,PO presense;
    class SM,SO sense;
    class LENSE lense;
    class DM,DO dispense;
    class MQTT,PG,TARGET system;
```

## Runtime Flow

```mermaid
sequenceDiagram
    participant Endpoint as Endpoint or Presense
    participant Sense as IoT Sense
    participant Broker as MQTT Broker
    participant Lense as IoT Lense
    participant Postgres as Postgres
    participant Dispense as IoT Dispense
    participant Target as Target system

    Endpoint->>Sense: Read values
    Sense->>Broker: Publish telemetry, status and errors
    Broker->>Lense: Deliver subscribed message
    Lense->>Postgres: Persist telemetry and health
    Lense->>Lense: Update dashboard and quick metrics
    Broker->>Dispense: Deliver subscribed message
    Dispense->>Target: Forward selected payload
```

## Deployment View

```mermaid
flowchart LR
    subgraph Host["Docker host"]
        subgraph Network["docker-compose network"]
            PM["presense-modbus-01..03<br/>ports 8301..8303 / 5020"]
            PO["presense-opcua-01..02<br/>ports 4840..4841"]
            SM["sense-modbus<br/>port 8100"]
            SO["sense-opcua<br/>port 8101"]
            MQTT["mqtt<br/>port 1883"]
            LENSE["lense<br/>port 8000"]
            PG[("postgres<br/>port 5432")]
            DM["dispense-modbus<br/>port 8200"]
            DO["dispense-opcua<br/>port 8201"]
            TARGET["dispense-broker<br/>port 1884"]
        end
    end

    PM -.-> SM
    PO -.-> SO
    SM --> MQTT
    SO --> MQTT
    MQTT --> LENSE
    LENSE --> PG
    MQTT --> DM
    MQTT --> DO
    DM --> TARGET
    DO --> TARGET

    classDef presense fill:#1b1f27,stroke:#8a9099,stroke-dasharray:5 5,color:#e7e9ee;
    classDef sense fill:#1b1f27,stroke:#14b8a6,color:#e7e9ee;
    classDef lense fill:#1b1f27,stroke:#ff8a1d,color:#e7e9ee;
    classDef dispense fill:#1b1f27,stroke:#ef4444,color:#e7e9ee;
    classDef system fill:#1b1f27,stroke:#ffd21f,color:#e7e9ee;

    class PM,PO presense;
    class SM,SO sense;
    class LENSE lense;
    class DM,DO dispense;
    class MQTT,PG,TARGET system;
```

## Extension Points

```mermaid
flowchart LR
    Source["Custom source endpoint"]
    Presense["Custom Presense module<br/>optional simulator"]
    Sense["Custom Sense module<br/>protocol collector"]
    Shared["Shared MQTT, topic and payload helpers"]
    Broker["MQTT Broker"]
    Lense["IoT Lense<br/>health and config visibility"]
    Dispense["Custom Dispense module<br/>outbound adapter"]
    Target["Target system"]

    Source -->|implement protocol reader| Sense
    Presense -.->|simulate protocol endpoint| Sense
    Sense -->|use topic and payload conventions| Shared
    Shared -->|publish| Broker
    Broker -->|observe| Lense
    Broker -->|subscribe| Dispense
    Dispense -->|implement target writer| Target

    classDef presense fill:#1b1f27,stroke:#8a9099,stroke-dasharray:5 5,color:#e7e9ee;
    classDef sense fill:#1b1f27,stroke:#14b8a6,color:#e7e9ee;
    classDef lense fill:#1b1f27,stroke:#ff8a1d,color:#e7e9ee;
    classDef dispense fill:#1b1f27,stroke:#ef4444,color:#e7e9ee;
    classDef system fill:#1b1f27,stroke:#ffd21f,color:#e7e9ee;
    classDef external fill:#1b1f27,stroke:#8a9099,color:#e7e9ee;

    class Presense presense;
    class Sense sense;
    class Lense lense;
    class Dispense dispense;
    class Broker,Shared,Target system;
    class Source external;
```

## OPC UA style subscriptions

The beta stack supports an OPC UA style subscription path between IoT Presense and IoT Sense. Presense exposes a streaming subscription endpoint and Sense can consume it with `readMode: "subscription"`.

This models subscription behavior for the demo stack. Security, certificates and encrypted real OPC UA sessions are planned later.

## Main Building Blocks

| Building block | Responsibility | Notes |
|---|---|---|
| IoT Presense | Simulates protocol endpoints | Used for demo and test scenarios only |
| IoT Sense | Reads source endpoints | Publishes telemetry, status and errors |
| MQTT Broker | Transports messages | Current default transport |
| IoT Lense | Shows health, quick metrics and configuration | Helps find the broken link quickly |
| Postgres | Stores telemetry and health data | Used by IoT Lense |
| IoT Dispense | Forwards selected messages | Sends data to target systems |
| Target systems | Receives forwarded data | External to OT.io |

## Real Machines and Presense Simulations

Real machines and Presense simulations are intentionally treated differently.

| Endpoint type | Shown in connection graph | Clickable | Visual style |
|---|---:|---:|---|
| Real external machine | Yes | No | Solid grey border |
| Presense simulation | Yes | Yes | Dashed grey border |
| OT.io component | Yes | Yes where applicable | Component color |

This keeps the graph useful without pretending that external machines are managed by OT.io.
