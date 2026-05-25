# Extension Readiness Review

This review captures the intended extension seams for the beta.

## Main rule

New source protocols belong in IoT Sense. New target systems belong in IoT Dispense. Lense should stay the central diagnostics and configuration surface.

## Sense

Sense is now split into:

```text
apps/sense/
├── main.go
└── internal/
    ├── app
    └── protocols
```

`internal/app` contains reusable runtime code:

- API routes
- configuration workflow
- worker lifecycle
- source health state
- MQTT publishing through a transport abstraction
- UI serving

`internal/protocols` contains protocol-specific code:

- `Reader`
- `Subscriber`
- protocol registry
- Modbus reader
- OPC UA HTTP reader/subscriber

## Current extension seams

| Seam | Interface | Purpose |
|---|---|---|
| Source protocol | `protocols.Reader` | Poll/read values from endpoints. |
| Source subscription | `protocols.Subscriber` | Stream values from endpoints. |
| Outbound transport | `app.EventPublisher` | Publish normalized messages. |
| Shared MQTT helper | `shared/mqttx` | Current MQTT transport implementation. |
| Shared UI | `shared/web` | Common chart and UI behavior. |

## Result

A new Sense protocol should not need to copy the whole Sense application. It should add a protocol reader/subscriber and register it.

A new outbound transport can be added behind `EventPublisher` without changing protocol readers.

## Test coverage added

- Registry built-in protocols
- Custom reader registration
- OPC UA HTTP read behavior
- OPC UA subscription behavior
- Generic value scaling for numbers and strings
- Static check that Sense uses an outbound transport abstraction
