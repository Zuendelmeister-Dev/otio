# Testing

Run all tests from the repository root with the wrapper scripts.

## Go tests

Windows:

```powershell
.\test.cmd
```

Linux and macOS:

```bash
bash scripts/test.sh
```

The repository is a multi-module Go workspace. A plain `go test ./...` from the root is not the canonical command; the scripts run each module, including `apps/plc4go-modbus`.

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
