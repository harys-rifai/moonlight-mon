# Moonlight Apps

Monitoring dashboard for your infrastructure — PostgreSQL, Redis, databases, servers, and network services.

Originally forked from **Beszel**, now rebranded as **Moonlight apps**.

## Features

- **PostgreSQL backend** — Runs on PostgreSQL instead of SQLite
- **Database monitoring** — Dedicated Databases page at `/databases` for PostgreSQL, MySQL, Redis, SQL Server, Oracle, DB2, Aerospike, and more
- **Network monitoring** — TCP port checks for any service on any host
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
cp .env.example .env   # if .env.example exists
# or edit .env directly
```

Example `.env`:

```ini
# Database connection (PostgreSQL)
DATABASE_URL=postgres://postgres@localhost:5006/server?sslmode=disable

# App URL
APP_URL=http://localhost:8090

# Environment mode (dev enables auto-migration and development build tag)
ENV=dev
```

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
└── migrations/               # Beszel-specific migrations
    ├── initial-settings.go   # Default app name: "Moonlight apps"
    └── ...
```

## Key Changes from Beszel

| Area | Change |
|------|--------|
| Name | Beszel → Moonlight apps |
| Database | SQLite → PostgreSQL |
| Frontend proxy | Vite dev server auto-started by `run.sh` |
| Databases page | New `/databases` route for database port monitoring |
| Navbar | Added Databases menu item |
| PostgreSQL extensions | `pgcrypto`, `nocase` collation, `uuid_generate_v7`, `hex`, `randomblob` pre-created before migrations |
| JSON functions | `json_extract` → `#>>` operator for PostgreSQL JSONB |
| Index syntax | Backticks → double quotes for PostgreSQL compatibility |

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

## Production Build

For production, build without the `development` build tag:

```bash
go build -o beszel-hub ./internal/cmd/hub
./beszel-hub serve --http=0.0.0.0:8090
```

Build the frontend for production:

```bash
cd internal/site
npm run build
```

## License

MIT
