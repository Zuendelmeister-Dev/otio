# IoT Sense Code Structure

Sense is built as a small runtime plus protocol-specific readers.

## Folder layout

```text
apps/sense/
├── main.go
├── static/
└── internal/
    ├── app/
    └── protocols/
```

## Runtime package

`internal/app` contains generic Sense behavior:

- API routes
- configuration workflow
- worker lifecycle
- source health handling
- metric payload mapping
- transport publishing
- UI route serving

This code should rarely need changes for a new source protocol.

## Protocol package

`internal/protocols` contains protocol-specific readers and subscribers.

To add a new source protocol:

1. implement `Reader`
2. implement `Subscriber` if the protocol supports subscriptions
3. register the implementation in `registry.go`
4. add configuration examples
5. add protocol tests
6. add the new source type to the documentation

The generic UI can usually remain unchanged because it reads normalized status, logs and metrics.


## Outbound transport seam

Sense currently publishes through MQTT, but the runtime uses the `EventPublisher` interface. This keeps protocol readers independent from the transport implementation.

Future publishers, for example Kafka, NATS or HTTP, should be implemented behind the same interface.
