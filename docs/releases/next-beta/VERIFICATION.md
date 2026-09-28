# Next beta verification

## UI release candidate — 27 September 2026

Branch: `codex/ui-usability-release`. This local record covers the working tree prepared for review; it does not claim a GitHub CI run.

| Check | Result |
|---|---|
| `scripts/test.ps1` | Passed: all nine Go modules, shared UI/chart tests, workspace navigation, deployment generation, Kubernetes consistency, live-topic selection and repository hygiene |
| `go vet ./...` in all nine Go modules | Passed |
| Examples 01 and 03 `docker compose ... config --quiet` | Passed; Docker CLI warned that its user config was inaccessible in the sandbox |
| Live Messages browser checks | Topic selection, persistent selection after refresh, paused filtering, empty results and directly visible escaped payload checked with local fixtures |
| Documentation | Six supplied screenshots incorporated; local Markdown links and image references pass repository check |
| Git hygiene | Candidate-file scan passed; private keys, certificate profile directories, environment overrides, image archives and publishing drafts excluded |
| Helm render/lint suites | Not rerun: Helm is unavailable on this session's PATH |
| Container builds / HA / physical devices / vulnerability scan | Not rerun for this UI candidate; earlier results below apply only to their dated runs |

Review changes include reusable topic selection logic with regression tests, rejecting aborted stale UI responses, an eight-second total deadline for live-message aggregation, explicit HTTP upstream errors, and updated repository checks in both test wrappers. Shared Go traffic counters remain bounded; UI rendering uses text content for message payloads. Go retains protocol/network logic, while JavaScript handles view filtering and interactions.

The repository scan is heuristic and is not a guarantee that all secrets are detected. Existing Go test suites skip optional external-broker tests when their endpoint variables are absent. No new coverage percentage or race-detector result is claimed.

## Historical verification — 19–20 September 2026

Checked locally on 2026-09-19; security update and follow-up checks on 2026-09-20, Windows/amd64, Go 1.27.1. This record covers the current working tree before its release commit; no GitHub CI run or release tag exists for these unpushed changes.

### Checks in the earlier run

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

These are the earlier run’s Go statement percentages, weighted across each module. They include newly added application code; they are not directly comparable with an older smaller package set. The overall historical baseline was 25.2%; this release candidate is 37.5% with a larger codebase. Integration tests running deployed binaries are not instrumented into these profiles.

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
