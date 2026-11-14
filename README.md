# Moonlight Apps

Monitoring dashboard for your infrastructure — PostgreSQL, Redis, databases, servers, and network services.

## Features

- **PostgreSQL backend** — Runs on PostgreSQL instead of SQLite
- **Database monitoring** — Dedicated Databases page at `/databases` for PostgreSQL, MySQL, Redis, SQL Server, Oracle, DB2, Aerospike, and more
- **Network monitoring** — TCP port checks for any service on any host
- **Telnet protocol support** — Monitor services using telnet protocol for legacy application compatibility
- **System monitoring** — CPU, memory, disk, temperature, bandwidth, GPU, load average, battery, containers, systemd, ZFS, SMART
- **Alerts** — Threshold-based alerts for metrics and network monitor failures
- **Realtime updates** — PostgreSQL LISTEN/NOTIFY for live dashboard updates
- **Multi-user auth** — Superusers, users, roles, OAuth2, MFA, OTP

## Quick Start

### Prerequisites

- PostgreSQL running on `localhost:5006` (configured in `.env`)
- Go installed
- Node.js and npm installed (for the frontend dev server)

### Environment

Copy the environment file and adjust as needed:

```bash
# If .env exists, it will be loaded automatically by run.sh
# On new devices without .env, run.sh auto-generates default credentials
cp .env .env.backup 2>/dev/null || true
# or edit .env directly
```

If `.env` does not exist, `run.sh` auto-applies defaults (PostgreSQL user `postgres`, password `Password09`, port `5006`, database `server`, timezone `Asia/Jakarta`), so a fresh clone runs without manual setup. `.env.example` documents every variable.

Example `.env`:

```ini
# Database connection (PostgreSQL)
DATABASE_URL=postgres://postgres:Password09@localhost:5006/server?sslmode=disable

# App URL
APP_URL=http://localhost:8090

# Environment mode (dev enables auto-migration and development build tag)
ENV=dev

# Dashboard and server timezone (Asia/Jakarta = WIB, UTC+7)
APP_TIMEZONE=Asia/Jakarta
TZ=Asia/Jakarta
```

### Timezone

All dashboard times, chart labels, logs, and cleanup schedules use `APP_TIMEZONE` (default `Asia/Jakarta`, WIB / UTC+7). Database records are stored in UTC and converted for display, so changing the timezone never rewrites stored data.

### Run the server

```bash
./run.sh serve --http=0.0.0.0:8090
```

`run.sh` will:
1. Load environment variables from `.env`
2. Install frontend dependencies if needed
3. Compile translations (`.po` → `.ts`)
4. Start the Vite dev server on `localhost:5173`
5. Start the Go backend on port `8090`

Access the dashboard at **http://localhost:8090**

### Network Monitoring Configuration

Configure response monitoring from this agent with the following settings:

- **System**: local-machine
- **Target**: 127.0.0.1
- **Protocol**: TCP (also supports Telnet for legacy applications)
- **Port**: 5008
- **Interval**: 30 seconds by each data table and column

Configure response monitoring from this agent:

- System: local-machine
- Target: 127.0.0.1
- Protocol: TCP
- Port: 5008
- Interval (seconds): 30 by each data table and column
- Additional protocol telnet to connect to database

### Login

Default superuser credentials (set via environment or auto-generated):

```ini
USER_EMAIL=harys@gmail.com
USER_PASSWORD=password
```

## Project Structure

```
Server/
├── run.sh                  # Start script (Go backend + Vite dev server)
├── push.sh                 # Git initialization and push script
├── .env                    # Environment configuration (not committed)
├── .gitignore              # Git ignore rules
├── go.mod                  # Go module definition
├── internal/
│   ├── cmd/hub/            # Hub entry point
│   │   └── hub.go          # Main hub logic with PostgreSQL extension setup
│   ├── hub/                # Hub server implementation
│   │   ├── server_development.go   # Vite dev proxy for development mode
│   │   ├── server.go             # Production server
│   │   └── collections.go        # Collection management
│   ├── alerts/             # Alerting system
│   │   └── alerts_systemd.go   # Systemd alerts with PostgreSQL json_extract fix
│   └── site/               # Frontend (React + Vite)
│       ├── src/
│       │   ├── components/
│       │   │   ├── navbar.tsx          # Top navigation with Databases menu
│       │   │   ├── logo.tsx            # App logo (Moonlight apps)
│       │   │   ├── routes/
│       │   │   │   ├── databases.tsx   # Databases page
│       │   │   │   ├── monitors.tsx    # Network monitors page
│       │   │   │   ├── home.tsx        # Systems overview
│       │   │   │   └── ...
│       │   │   └── network-monitors-table/   # Monitor table component
│       │   ├── lib/
│       │   │   ├── api.ts              # PocketBase API client
│       │   │   └── utils.ts            # Utility functions
│       │   └── locales/en/en.ts        # English translations
│       └── vite.config.ts              # Vite configuration
├── local_pocketbase/       # PocketBase fork (embedded)
│   ├── core/
│   │   ├── collection_model.go     # Collection JSON unmarshal fix
│   │   ├── collection_import.go    # Import preserves auth options
│   │   └── collection_record_table_sync.go   # JSON validation fix
│   ├── tools/dbutils/
│   │   ├── index.go            # Backtick to double-quote fix
│   │   └── json.go             # json_valid removal for PostgreSQL
│   └── migrations/           # Database migrations
└── migrations/           # Database migrations
    ├── initial-settings.go   # Default app name: "Moonlight apps"
    └── ...
```
```

## Database Monitoring

The Databases page (`/databases`) shows network monitors filtered by common database ports:

| Database | Default Port |
|----------|-------------|
| PostgreSQL | 5432 (or 5006 in this setup) |
| MySQL / MariaDB | 3306 |
| Redis | 6379 |
| SQL Server | 1433 |
| Oracle | 1521 |
| DB2 | 50000 |
| Aerospike | 3000 |

Add monitors via the **Monitors** page or directly in the Databases table using TCP protocol.

## Database Observability

Adopted from `obs.md`: a collector-driven module for database health, capacity, and alerting.

- **Page** — `/db-health` (navbar "Database Health").
- **Inventory** — `db_inventory` stores name, application, environment, engine, host/port, database, credentials, owner, HA mode, cloud provider, group. Passwords are stored in a hidden field and never returned by the API.
- **Collector** — runs every 60s in the hub and records into `db_metrics`: availability, version, uptime, connections (total/active/idle/max/usage), long-running queries (>5 min), blocked sessions, lock wait, replication lag, database size, oldest transaction.
- **Alerting** — `db_alert_rules` holds warning/critical thresholds (defaults below); `db_alerts` follows the `new → warning → critical → acknowledged → resolved` lifecycle and can be acknowledged/resolved from the page.
- **Notifications** — transitions are sent to the webhooks and email addresses configured in admin user settings (Teams/Slack/Telegram via shoutrrr URLs, plus SMTP email).
- **Retention** — raw samples are pruned after 30 days.

| Metric | Warning | Critical |
|--------|---------|----------|
| connection_usage | 80% | 90% |
| long_queries | 1 | 5 |
| blocked_sessions | 10 | 25 |
| lock_wait_sec | 60 | 120 |
| replication_lag_sec | 60 | 300 |
| oldest_tx_sec | 300 | 900 |
| down | - | any |

PostgreSQL targets are collected natively. MySQL, SQL Server, and Oracle are supported as inventory entries today; their SQL collectors are the next phase from `obs.md` (MySQL also needs a driver dependency).

### Deep PostgreSQL metrics

Each cycle also records TPS/QPS, slow queries (>1 min and >5 min), blocked sessions, lock waits, idle-in-transaction, deadlocks rate, cache hit ratio, WAL bytes/s, temp bytes/s, block read/write time, and statement samples in `db_query_samples` (7-day retention).

Statement-level APM uses `pg_stat_statements` when installed; otherwise the collector falls back to the currently running statements slower than 5s. To enable full APM:

```sql
ALTER SYSTEM SET shared_preload_libraries = 'pg_stat_statements';
-- restart PostgreSQL, then in each database:
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
```
