# UI configuration and connection graphs

Lense, Sense, Dispense and the legacy Presense UIs share a scrollable graph viewport. Drag the graph background horizontally or vertically, use the scrollbars, or focus the viewport and use the left/right arrow keys. Lense groups sources, collection, messaging, storage/forwarding and destinations into lanes; Lense sits above MQTT and connects directly to Postgres. Sense uses equally spaced source, collector and broker columns.

System Components groups components by role and displays broker/database connection details as compact facts. Connection charts collect actual observations while the page is open, retain up to 300 samples in browser storage and fit the observed time range. The first observation is necessarily a single point; subsequent polls extend the line, even when the state is unchanged. Gaps longer than two minutes are not connected. History is browser-local, not server-side monitoring history.

## Forms and JSON

Open **Configuration → Form editor · protocol-specific fields**. Form edits update the existing JSON draft; editing JSON also refreshes the open form. Source protocol selection changes the connection and metric address presets. Expand **metrics** and review every address, scale and unit before applying: presets cannot infer a device's register map. Existing custom fields remain in the draft. Validation and application still use each application's existing workflow. Dispense's current configuration screen prepares drafts; its runtime write-back remains unavailable.

Sense namespace status/error topics show an explicit information message and clear the previous metric chart. Only metric topics display numeric history.

## Presense

The Modbus and legacy OPC UA HTTP demo instances expose `GET /api/config` and `PUT /api/config`. The form changes generator mode and base values in the running process. Choose **Validate changes**, review the proposed JSON, then **Apply proposal**. Background status refreshes do not overwrite drafts. If `OTIO_CONFIG_WRITE_TOKEN` is set, enter it in the token field. Unknown fields, invalid modes, out-of-range values and changes to the fixed listener protocol are rejected by the server.

Set `PRESENSE_CONFIG_PATH` to save settings across restarts. Example 01 mounts an individual configuration volume per simulator. With no path, settings last only until process restart. Host and listener ports still come from startup environment variables. **Host reachable from Sense** changes the copied example, not the listener.

**Copy Sense source** exports this instance's actual register/node mapping, scaling and port. Add the object to Sense's `sources` array, using a hostname reachable from the Sense process (Compose service names inside the stack).

The protocol selector also opens Protocol Lab for native Modbus TCP, RTU-over-TCP, OPC UA Binary, MQTT and AMQP simulation. Legacy instances keep their fixed protocol. In the Lab, choose **Simulated protocol**; the connection, address and Sense source example adapt to it. Native listeners run concurrently and share the temperature generator. MQTT/AMQP require configured broker publishers; AMQP credentials in the example must match the deployed broker. Other catalog adapters read external devices and do not claim to provide simulators.

## PostgreSQL data explorer

Open **Agents → System Components → Postgres**, or <http://localhost:8000/#agents/postgres>. The same explorer is available below **System Components**. It uses Lense's existing historian database connection; no extra database client, container or login is needed. It inherits access to Lense's UI/API and does not introduce a separate authentication boundary.

Select a public table, inspect its column names/types, and choose **Preview table**, or edit a query:

```sql
SELECT ts, agent_id, metric_name, metric_value
FROM metric_events
WHERE agent_id = 'modbus-machine-01' AND metric_value > 25
ORDER BY ts DESC
LIMIT 200;
```

Supported syntax is intentionally limited to `SELECT` columns or `*`, one table in `public`, comparisons (`=`, `!=`, `<>`, `<`, `>`, `<=`, `>=`, `LIKE`, `ILIKE`, `IS [NOT] NULL`) joined with `AND`, column-based `ORDER BY`, and `LIMIT`. Values are bound as parameters. Joins, arbitrary functions, comments, CTEs, additional statements and writes are rejected. The server also uses a read-only transaction and a four-second PostgreSQL statement timeout within a five-second request deadline.

The default limit is 200 rows, the maximum 500, and result data is capped at 2 MiB. Truncated results are labelled. **Export displayed rows · CSV** exports exactly the current result, including column headers; it does not export the entire database. Nulls become empty CSV fields; spreadsheet formula-like text is prefixed with an apostrophe. Use narrower filters for extracts. Editing the query clears the old result and disables export until the next successful read.

Result columns keep readable widths and scroll horizontally inside the table. Headers stay visible when scrolling down. Long text is shortened visually; click a text cell to open its complete value, with JSON formatted over multiple lines. Close the detail view with **Close** or Escape. CSV exports retain the complete values, including text shortened in the table.

## Example 01 startup

The PostgreSQL data explorer is integrated in Lense; no separate database-client container is needed.

From the repository root:

```powershell
docker compose -f examples/01-local-docker-compose/docker-compose.yml up --build -d
```

Open Lense at <http://localhost:8000>, then **Protocol Lab**, or use <http://localhost:8000/protocols>. Example 01 now includes Protocol Lab and RabbitMQ. If the Lab is unavailable, the Lense route displays a service-unavailable page with startup instructions. Reload existing browser tabs after rebuilding to load the shared UI scripts.
