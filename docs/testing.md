# Testing

## UI regression checks

`node scripts/ui-fixture-server.cjs` serves deterministic Lense (:18000) and Sense (:18100) test data without Docker. These fixtures are for layout and interaction checks, not evidence of live MQTT/database connectivity. Verify graph dragging, grouped component facts, status history after multiple polls, protocol form changes mirrored into JSON, and switching a metric preview to `_sense/status` without leaving a stale chart.

For live Presense checks, run its Go module with a dedicated `PRESENSE_UI_PORT` and `MODBUS_PORT`, apply a constant value through the form, and verify `/api/values` and a Modbus read. Configuration API tests cover token rejection, invalid settings, persistence/reload and exported Sense mappings. Shared JavaScript tests cover protocol presets, history retention, repeated unchanged observations and denied browser storage.

Run all tests from the repository root with the wrapper scripts.

Use Go 1.27.1 or newer and Node.js 24. The workspace selects Go 1.27.1; its first use may download that toolchain. Keep the patch release current for standard-library security fixes.

## Go and JavaScript tests

Windows:

```powershell
.\test.cmd
```

Linux and macOS:

```bash
bash scripts/test.sh
```

The repository is a multi-module Go workspace. A plain `go test ./...` from the root is not the canonical command; the scripts run each module, including `apps/plc4go-modbus`.

Both runners stop on a failed command and can be invoked from outside the repository by their absolute path.

## Coverage

On Windows run `.\coverage.cmd`. With PowerShell 7 on other platforms, run `pwsh -File scripts/coverage.ps1`. Reports are generated in the ignored `coverage/` directory:

- `coverage.html`: annotated source coverage
- `merged.coverage.txt`: function and total statement coverage
- per-module profiles and summaries

The [September 2026 review](quality/review-2026-09.md) records the baseline and remaining integration-test gaps. Tests that search HTML source for strings do not establish browser behavior or Go execution coverage.

## Static analysis, race detection and vulnerabilities

From a POSIX shell at the repository root:

```bash
for module in apps/* shared/mqttx; do
  (cd "$module" && go vet ./... && go test -race ./...) || exit 1
done
```

Race detection requires CGO and a C compiler. The CI workflow runs it on Linux.

The vulnerability scanner needs a module directory, even when scanning the full workspace:

```bash
cd apps/lense
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ../presense-modbus/... ../presense-opcua/... ../sense/... ../lense/... ../dispense/... ../plc4go-modbus/... ../protocol-lab/... ../ha-agent/... ../../shared/mqttx/...
```

This checks the selected platform and current vulnerability database; it is not a container or deployment audit. The GitHub Actions workflow runs Go/JavaScript tests on Windows and Linux, uploads coverage, and runs vet, race detection, vulnerability scanning and demo image builds on Linux.

## JavaScript tests

```bash
node shared/web/iot-ui.test.js
node shared/web/standard-chart.test.js
node shared/web/workspace-ui.test.js
```

## Docker build test

```bash
docker compose build --no-cache
```

## Current quality goal

Every exported function should have direct unit tests or an explicit reason why it is covered by integration tests instead.

The current beta still contains test gaps. Track them in:

- `docs/quality/function-test-inventory.md`

## Kubernetes

The normal test scripts check Kubernetes source/topology consistency with Node.js. Render the example with `kubectl kustomize examples/05-kubernetes`. A separate CI job validates against a temporary kind API server, builds/loads images and runs `bash scripts/kubernetes-smoke.sh` to check native protocols, broker samples, five healthy sources and persisted telemetry. The smoke test requires kubectl, Bash and Node.js. See [Example 05](../examples/05-kubernetes/README.md).

## Helm

With Helm on PATH, run `node scripts/helm-chart.test.js`. This checks linting, rendered resource counts, source/topology consistency, custom images, registry secrets, AMQP credential escaping, storage classes, PVC retention and invalid values. CI also installs [Example 06](../examples/06-helm/README.md) and runs its `helm test` connectivity hook. The separate HA chart was also installed and tested locally; see the [Example 07 validation record](../examples/07-kubernetes-ha/VALIDATION.md). This does not establish a local live installation of Example 06.

The separate `Kubernetes HA example` workflow runs Example 07 on a four-node kind cluster. Its PowerShell failover script verifies application Lease takeover, a single ready endpoint, database primary replacement and resumed telemetry after application/broker/database failures. It does not claim packet-loss, network-partition or physical-host fault coverage. Run `node scripts/ha-chart.test.js` for the fast Helm checks.

## Release verification

The [next beta verification record](releases/next-beta/VERIFICATION.md) distinguishes current unit/static checks from previously completed live checks and remaining CI-only coverage.
