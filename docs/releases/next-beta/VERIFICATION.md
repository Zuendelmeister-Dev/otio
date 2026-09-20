# Next beta verification

Checked locally on 2026-09-19; security update and follow-up checks on 2026-09-20, Windows/amd64, Go 1.27.1. This record covers the current working tree before its release commit; no GitHub CI run or release tag exists for these unpushed changes.

## Current final checks

| Check | Result |
|---|---|
| `scripts/test.ps1` | Passed: nine Go modules, three shared JavaScript suites, Kubernetes configuration checks |
| `scripts/coverage.ps1` | Passed; HTML and per-module profiles generated in ignored `coverage/` |
| `go vet ./...` in every Go module | Passed |
| `scripts/helm-chart.test.js` | Passed: existing chart lint/render/configuration checks |
| `scripts/ha-chart.test.js` | Passed: HA chart lint/render/leader Service selection and topology checks |
| Compose configuration, Examples 01 and 03 | Passed |
| `git diff --check` | Passed |
| Go vulnerability scan | Passed after upgrading amqp091-go to 1.13.0: no vulnerabilities found by govulncheck 1.8.0 on Windows/amd64 |

The Go test run skips optional real-broker tests when their endpoint environment variables are absent. Passing unit tests does not establish physical-device interoperability, live browser behavior or container security.

## Statement coverage

These are current Go statement percentages, weighted across each module. They include newly added application code; they are not directly comparable with an older smaller package set. The overall historical baseline was 25.2%; this release candidate is 37.5% with a larger codebase. Integration tests running deployed binaries are not instrumented into these profiles.

| Module | Coverage |
|---|---:|
| Presense Modbus | 42.9% |
| Presense OPC UA HTTP demo | 36.5% |
| Sense | 43.7% |
| Lense | 27.0% |
| Dispense | 35.1% |
| PLC4Go Modbus demo | 29.0% |
| Protocol Lab | 50.8% |
| HA supervisor | 25.2% |
| Shared MQTT | 97.8% |
| **Workspace total** | **37.5%** |

## Live checks already completed for this candidate

The [HA validation record](../../../examples/07-kubernetes-ha/VALIDATION.md) records local Linux image builds, a four-node kind deployment, completed application rolling updates and successful Sense/Lense/broker/PostgreSQL failover exercises. The final exercise checked fresh rows after each recovery. A graceful PostgreSQL primary deletion took about three minutes. Both application pairs ended at 2/2 Ready, with one active Service endpoint each; all three broker and database replicas recovered.

Earlier browser checks in this work verified the PostgreSQL viewer against 200 real metric rows, horizontal scrolling, readable row heights and the full JSON/value dialog. These were manual checks, not an automated browser test suite. Existing unit/native protocol tests cover additional API, parser, configuration and wire-protocol behavior.

## Limits and pending remote checks

- Linux race-detector and remote CI results remain pending; this final local run did not execute a race detector. The CI workflows include those checks.
- The Example 07 local deployment does not prove live Example 05/06 smoke tests; those have separate CI jobs.
- No physical PLC validation, ARM hardware rollout, partition-fencing test, physical-node fault test, backup-restore exercise, image vulnerability audit or lossless-delivery guarantee is claimed.
- Remaining low runtime/API unit coverage and HTML-source assertions are explicit limitations; coverage is not a release-readiness score.

## Security follow-up — 2026-09-20

The initial release scan identified [GO-2026-6372](https://pkg.go.dev/vuln/GO-2026-6372) in `github.com/rabbitmq/amqp091-go` 1.10.0. Protocol Lab now pins fixed version 1.13.0. Its standalone module tests and `go vet` passed; the subsequent complete workspace scan reported no vulnerabilities. This is a platform-specific source scan, not an image audit. Previously built images must be rebuilt to contain the fix.

A new live-broker test and image build for this dependency update could not yet be completed because Docker Desktop did not respond during the local startup attempt. The existing HA test results remain valid for their documented run; the HA applications do not use this AMQP library. CI includes the Protocol Lab image build and actual MQTT/AMQP round trips.
