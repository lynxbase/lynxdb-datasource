# LynxDB data source for Grafana

Query and visualize [LynxDB](https://github.com/lynxbase/lynxdb) logs and metrics
in Grafana using Lynx Flow / SPL2. This is a backend data source plugin
(Go backend + React frontend) that talks to the LynxDB REST API.

## Features

- Lynx Flow / SPL2 query editor with field and value autocomplete and live parse
  validation (powered by the LynxDB `/fields` and `/query/explain` endpoints).
- Logs and metrics in one editor: `Logs` renders events in the Explore Logs view;
  `Metrics` renders `timechart` results as time series.
- Log-volume histogram in Explore, grouped by log level.
- Live tail in Explore over the LynxDB SSE stream.
- Annotation queries: log lines become dashboard annotations.
- Template variables: `fields`, `sources`, and `values(<field>)`.
- Secure bearer-token auth stored encrypted in `secureJsonData`.
- Bundled dashboards for Nginx access logs and PostgreSQL (import from the
  plugin's Dashboards tab). Each has a `$source` text variable to point at your
  log source.

## Configuration

| Setting        | Description                                              |
| -------------- | -------------------------------------------------------- |
| URL            | LynxDB base URL, e.g. `http://localhost:3100`            |
| API token      | Bearer token (only needed when LynxDB auth is enabled)   |
| Default index  | Optional default index / namespace                       |
| Time field     | Event timestamp field (default `_time`)                  |
| Message field  | Field shown as the log line body (default `_raw`)         |
| Level field    | Field used for log level and volume grouping (default `level`) |
| Max lines      | Default maximum log lines per query (default `1000`)     |

Provisioning example: see [`provisioning/datasources/datasources.yml`](./provisioning/datasources/datasources.yml).

## Querying

The editor sends the query text to `POST /api/v1/query`.

```text
# logs
_source=nginx | where status >= 500

# metrics (time series)
* | timechart count() span=1m by level
```

Template variable queries (Dashboard settings > Variables > Query):

```text
fields            # all field names
sources           # all source names
values(level)     # distinct values of the level field
```

## Development

Requires Go and an LTS Node.js.

```bash
npm install            # frontend deps
npm run build          # build frontend (webpack)
mage -v                # build backend for all platforms
npm run server         # start Grafana with the plugin (Docker)
```

Run a local LynxDB with generated data to develop against:

```bash
lynxdb demo            # in-memory server on :3100 with sample logs
```

### Tests

```bash
npm run test:ci                                  # frontend (Jest)
mage test                                        # backend unit tests
LYNXDB_TEST_URL=http://127.0.0.1:3101 mage test  # backend integration tests
```

The integration tests are skipped unless `LYNXDB_TEST_URL` is set; point it at a
running LynxDB (for example the address printed by `lynxdb demo`).

## Distributing the plugin

Community plugins must be signed before Grafana will load them outside of a
development environment. See the Grafana docs on
[signing](https://grafana.com/developers/plugin-tools/publish-a-plugin/sign-a-plugin)
and [publishing](https://grafana.com/developers/plugin-tools/publish-a-plugin/publishing-and-signing-criteria).
The bundled GitHub Actions [release workflow](./.github/workflows/release.yml)
builds, signs, and packages the plugin when a version tag is pushed.
