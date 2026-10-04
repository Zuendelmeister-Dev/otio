# Historian retention, freshness and recovery

Lense stores measurements in `metric_events`, current source status in `agent_status`, and errors in `error_events`. A connected collector can still contain an old measurement; connection health and data freshness answer different questions. The PostgreSQL viewer is a limited data browser. Its CSV export and `/api/export` are bounded extracts, not database backups.

## Optional retention

Retention is **disabled by default**. Restart Lense with explicit retention settings to remove old measurements and errors automatically. No UI or HTTP endpoint deletes historian data.

| Environment variable | Default | Allowed values and behavior |
|---|---:|---|
| `LENSE_RETENTION_METRIC_DAYS` | `0` | `0` keeps all measurements; `1..36500` keeps the selected age in days |
| `LENSE_RETENTION_ERROR_DAYS` | `0` | `0` keeps all errors; `1..36500` keeps the selected age in days |
| `LENSE_RETENTION_INTERVAL_SECONDS` | `3600` | `300..86400`; first cleanup runs after this interval, not immediately at startup |
| `LENSE_RETENTION_BATCH_SIZE` | `1000` | `1..10000` rows per transaction |
| `LENSE_RETENTION_MAX_BATCHES` | `5` | `1..20` batches per table per cleanup |
| `LENSE_STALE_AFTER_SECONDS` | `300` | `30..86400`; measurement freshness warning threshold |

For example, save this Compose override as `compose.historian.yml` at the repository root:

```yaml
services:
  lense:
    environment:
      LENSE_RETENTION_METRIC_DAYS: "30"
      LENSE_RETENTION_ERROR_DAYS: "90"
      LENSE_STALE_AFTER_SECONDS: "300"
```

Apply it together with Example 01:

```sh
docker compose -f examples/01-local-docker-compose/docker-compose.yml -f compose.historian.yml up -d lense
```

Back up valuable data before enabling or shortening retention. Retention uses the stored measurement/error timestamp `ts`, with an exclusive cutoff: a row at the cutoff is kept. `agent_status`, configuration and certificates are not deleted. The default limits remove at most 5,000 rows from each enabled table per run. A large backlog is drained over successive runs. Each batch has a five-second statement timeout and one-second lock timeout; the entire run has a 30-second deadline. Failed batches roll back, while earlier committed batches remain applied. Database advisory locks serialize batches across Lense processes. Retention is not HA fencing for collectors.

The schema migration adds timestamp indexes for bounded deletion. Existing large installations should allow time and disk space for those initial index builds. Ordinary deletion frees PostgreSQL space for reuse after vacuuming; it does not immediately shrink the filesystem or PVC. This feature does not execute `VACUUM FULL`, drop tables, or truncate history.

Use `GET /api/historian/status` to inspect database availability, configured limits and the most recent cleanup counts/warning. Invalid environment values fail startup with the relevant variable name instead of silently activating a different policy. Missing or zero retention settings retain the complete history.

## Measurement freshness

`GET /api/historian/freshness` reports the newest stored measurement for each source in the current topology. The lookup uses the existing per-source timestamp index. At most 500 sources are included in one response; `truncated: true` means remaining sources need individual requests.

For one measurement series, request:

```text
/api/historian/freshness?agent_id=modbus-machine-01&metric_name=temperature
```

| Status | Meaning |
|---|---|
| `fresh` | The last measurement is within the configured time window |
| `stale` | The last measurement is historical; check the source, collector and MQTT connection |
| `missing` | No measurement is stored for this source/series yet |
| `clock-skew` | The measurement is more than one minute in the future; check device and collector clocks |
| `unknown` | The database query failed; the HTTP response is `503` and freshness cannot be verified |

Age is calculated from the message's **measurement timestamp**, not the time an HTTP page was opened or a broker status was received. A source summary describes its newest metric; it does not guarantee every metric is fresh. Use the metric-specific request to check slow or independently updated signals. Differences up to one minute in the future are tolerated as clock drift and displayed with an age of zero; larger differences produce `clock-skew`. The API preserves the last timestamp and age, alongside a plain warning; it never substitutes zero for a failed query.

## Back up a Compose deployment

Requires **Python 3.10 or newer** and Docker Compose. The `postgres` service must already be running. PostgreSQL client tools run inside its existing container; no extra database-client image is installed. Commands below run from the repository root and use only relative repository paths.

Windows PowerShell:

```powershell
./scripts/historian.ps1 backup --compose examples/01-local-docker-compose/docker-compose.yml --output backups/otio-before-upgrade.dump
```

Linux/macOS:

```sh
sh scripts/historian.sh backup --compose examples/01-local-docker-compose/docker-compose.yml --output backups/otio-before-upgrade.dump
```

Direct Python invocation is also available on all platforms:

```sh
python scripts/historian.py backup --compose examples/03-protocol-lab/docker-compose.yml --output backups/protocol-lab.dump
```

Set `--service` for another database service name. The tool uses that container's `POSTGRES_USER`, `POSTGRES_DB` and `POSTGRES_PASSWORD`; it does not print or add passwords to command arguments. `--database` selects a different plain database name.

Backups are PostgreSQL custom-format archives, produced by `pg_dump` from a consistent database snapshot while Lense can keep running. Binary output is streamed to a private temporary file. Only a successfully completed archive is published, atomically and without overwriting an existing backup. A failed dump leaves no completed backup file. Choose a local filesystem with hard-link support (such as NTFS, ext4 or APFS); an unsupported network filesystem reports an error. Keep backups outside the application data volume and protect their contents. Global PostgreSQL roles, application configurations, certificate profiles and deployment secrets are separate backup items.

## Restore into an empty database

Restore is performed only when you explicitly run the command. The tool requires the exact target name in `--confirm-database`, checks that the archive can be read, and refuses a target containing user tables, views, sequences, functions, types, extra schemas or extensions. It never runs `--clean` or drops existing tables. `pg_restore --single-transaction --exit-on-error` rolls back a failed restore.

For Example 01 with its default `iot` database user, prepare a separate recovery database:

```sh
docker compose -f examples/01-local-docker-compose/docker-compose.yml exec -T postgres createdb -U iot otio_recovered
```

Use your configured database user instead of `iot` if changed. Keep Lense stopped while switching to the recovered database so its automatic schema migration does not populate the target before restoration.

```powershell
./scripts/historian.ps1 restore --compose examples/01-local-docker-compose/docker-compose.yml --database otio_recovered --input backups/otio-before-upgrade.dump --confirm-database otio_recovered
```

On Linux/macOS, replace `./scripts/historian.ps1` with `sh scripts/historian.sh`. Select `otio_recovered` in Lense's PostgreSQL connection settings and restart Lense, then verify retained row counts and fresh measurements. Keep the original database available until the recovery has been verified. Archive ownership and grants are not restored; the connected database user owns the restored objects. Use archives you trust: PostgreSQL restores execute SQL from the archive. The tool limits an individual client operation to 15 minutes; larger installations may need a separately managed backup/restore procedure.

## Native PostgreSQL

Requires Python and PostgreSQL's `pg_dump`, `pg_restore`, `psql` and, when creating a recovery database, `createdb` on PATH. Use a compatible PostgreSQL client: `pg_dump` must support the server's major version and `pg_restore` must support the archive version.

Configure standard libpq variables; the tool does not interpret `POSTGRES_DSN` or include credentials in a connection URI argument. Use a private password file via `PGPASSFILE` or your existing libpq password file; `PGPASSWORD` is also supported. Set `PGSSLMODE` and certificate variables to match your server's TLS requirements.

```powershell
$env:PGHOST = "127.0.0.1"
$env:PGPORT = "5432"
$env:PGUSER = "iot"
$env:PGDATABASE = "iotdb"
./scripts/historian.ps1 backup --native --output backups/native.dump
createdb --maintenance-db=postgres otio_recovered
./scripts/historian.ps1 restore --native --database otio_recovered --input backups/native.dump --confirm-database otio_recovered
```

For a POSIX shell, export the same `PG*` variables and use the shell wrapper. Native mode requires a plain `PGDATABASE` name or `--database`; use separate `PG*` variables rather than embedding a password in that name. For Kubernetes, the native mode works with a PostgreSQL port-forward and matching connection variables; the scripts do not infer CloudNativePG credentials or replace its operator-managed backup/recovery workflows.

## Verification

Run the isolated tests without a live database:

```sh
python -B -m unittest discover -s scripts -p historian_test.py
```

The Go suite covers retention bounds, cutoffs, advisory locks, transactional failures, source/metric scoping, freshness thresholds, clock skew and explicit database failures. The Python suite covers binary output, incomplete/competing backups, target confirmation, nonempty-target refusal, credential redaction and transactional restore arguments.

Optional real tests require an explicitly chosen **disposable PostgreSQL test server**. Python creates and removes two uniquely named test databases, so the role needs `CREATE DATABASE`. Go creates and removes one unique test schema, leaving public historian tables untouched. Existing application data is not used as a fixture.

```powershell
$env:OTIO_HISTORIAN_TEST_COMPOSE = "examples/03-protocol-lab/docker-compose.yml"
python -B -m unittest discover -s scripts -p historian_test.py
```

For a native test server, set `OTIO_HISTORIAN_TEST_NATIVE=1` with your `PG*` variables instead. To run the Go PostgreSQL integration check, set `LENSE_TEST_POSTGRES_DSN` to the disposable test database connection string and run `go test ./...` from `apps/lense`. Keep private credentials out of committed scripts/configuration. Without these variables, live checks skip explicitly. Passing isolated tests does not establish a completed live recovery or backup storage policy.
