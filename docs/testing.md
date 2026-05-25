# Testing

Run all tests from the repository root.

## Go tests

```bash
go test ./...
```

## JavaScript tests

```bash
node shared/web/iot-ui.test.js
node shared/web/standard-chart.test.js
```

## Docker build test

```bash
docker compose build --no-cache
```

## Current quality goal

Every exported function should have direct unit tests or an explicit reason why it is covered by integration tests instead.

The current beta still contains test gaps. Track them in:

- `docs/quality/function-test-inventory.md`
