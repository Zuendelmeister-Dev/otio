# Implementation and verification — 4 October 2026

This records the local changes following the [repository review](review-2026-10-04.md), starting from commit `ae262b2905ac9549dfc990c2ab75fdd507904018` on `feature/ui-usability-release`. It describes the working tree, not a new remote CI result. The existing static, Go-served UI is retained as requested; no Next.js migration or Node server was added.

## Four agreed improvements

| Area | Result |
|---|---|
| Protocol consistency | `shared/protocolmeta/catalog.json` supplies Sense registration/validation, Lab capabilities, browser presets and the generated Sense support table. Actual Reader/Subscriber support determines allowed read modes. UDP IP adapters remain selectable in Lab. |
| Configuration and failure behavior | Instance ownership and draft revisions prevent stale responses or proposals from applying the wrong configuration. Sense saves atomically, rejects concurrent/stale mutations and reports background reconnection separately from successful saving. Unready prior MQTT clients are disconnected on restart. HA supervision continues after activation and stops hung/unhealthy children within configurable budgets. |
| Historian maintenance | Optional bounded retention, timestamp indexes, source/metric freshness checks and portable PostgreSQL custom-format backup/restore helpers. Automatic deletion is off by default. Restore requires explicit target confirmation and an empty database. |
| Dispense outputs | A common output writer supports existing MQTT forwarding and HTTP/HTTPS POST. Bounded queues, retry classifications, stable per-message idempotency keys and acceptance/failure/uncertainty counters make delivery limits visible. |

Related corrections include MQTT open-socket readiness, exact hierarchical topic-prefix matching, source quality propagation, OPC UA demo subscription aliases, AMQP simulator isolation, component probe validation, correctly escaped CSV extracts and ARM source archives containing every Docker COPY input. Ansible redeployment preserves runtime configuration and environment secrets unless the documented replacement flags are selected.

## Everyday interface

The shared interface now has clear page titles, readable navigation, a consistent dark appearance, keyboard focus, restrained colors and responsive forms. It stays dark regardless of operating-system appearance or earlier browser theme preferences; there is no appearance selector. Measurement freshness has its own warning and a link to the affected device. The status page explains data retention and links to backup instructions. Common form fields have readable labels and brief hints; advanced JSON remains available.

Graph selection and dialog focus survive automatic refreshes. Unrelated flows fade when a node is hovered or keyboard-focused. The [screenshot gallery](../ui-workflows.md#current-interface-screenshots) and root README show the updated interface with explicitly labelled sample data; earlier deployed-stack screenshots remain in the gallery.

## Verification

| Check | Local result |
|---|---|
| `scripts/test.ps1` | Passed across ten Go modules and included JavaScript, generated-catalog, configuration, packaging, deployment and repository checks |
| `go vet ./...` in all ten modules | Passed |
| Config response-order behavior | Deferred responses exercise instance switching, editing after validation, duplicate apply, reset and history ownership |
| MQTT failure behavior | Local wire-level broker tests cover reconnect state, socket loss, uncertain delivery and replacement of an unready publisher |
| Sense save/reconnect behavior | A local broker withholding MQTT CONNACK verifies prompt save acceptance followed by a distinct reconnecting state |
| HA failure behavior | Deterministic watchdog checks plus a real child process and hanging HTTP health endpoint verify termination on failure |
| ARM deployment archive | An actual archive is extracted into a fresh directory; required app Docker COPY inputs are checked |
| Python backup/restore helper | Nine unit tests passed; the optional real-PostgreSQL integration test was explicitly skipped |
| Kubernetes Example 05 | `kubectl kustomize` rendered successfully |
| Browser | The final dark overview and device dialog, removed appearance selector, keyboard route focus and 375-pixel overview width were checked with sample API data; no console errors. Earlier settings/status, focus-after-refresh and responsive checks also passed. |
| Formatting/privacy checks | `git diff --check` passed; application, shared, deployment, script and documentation sources contain no local user paths or machine-specific username |

Docker's daemon was unavailable, so new container builds, a live Compose/cluster rollout, real database backup/restore and hardware interoperability were not exercised locally. Helm was unavailable on PATH. Race detection could not run locally without a C compiler; the Linux CI workflow includes race, vet, broker and deployment checks. A fresh vulnerability scan was not executed locally; CI's scanner now includes the new protocol module. Existing coverage percentages in the review were not recomputed for this implementation.

## Operational limits

MQTT QoS 0 confirms a local transport handoff, not broker receipt. An ambiguous publish failure is counted separately and is not blindly retried. HTTP 2xx means the receiver accepted the POST; retries can still duplicate a message unless the receiver honors its idempotency key. Dispense queues remain in memory and are not durable across process restart. These are explicit limits, not lossless-delivery guarantees.

Freshness is based on stored measurement timestamps. A source summary checks its newest metric; individual metric checks remain necessary for independently updated signals. Retention is opt-in, removes old data in bounded batches and does not immediately shrink the database volume. Backups cover the selected database; configurations, certificate profiles and deployment secrets require separate copies. See [historian operations](../historian.md) and [Dispense delivery rules](../dispense.md#delivery-retries-and-duplicates).
