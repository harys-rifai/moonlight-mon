#!/usr/bin/env bash
# Run the Beszel Hub server locally against the configured PostgreSQL database.
#
# Required environment (set in .env):
#   DATABASE_URL - PostgreSQL connection string, e.g.
#                  postgres://postgres@localhost:5006/server?sslmode=disable
#
# Usage: ./run.sh [extra args...]
#   Extra args are forwarded to the beszel binary, e.g. ./run.sh serve --http=0.0.0.0:8090

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Load environment variables from .env (if present)
if [ -f "$SCRIPT_DIR/.env" ]; then
    set -a
    # shellcheck disable=SC1091
    . "$SCRIPT_DIR/.env"
    set +a
fi
# Set default DATABASE_URL if not set (for new devices without .env)
if [ -z "${DATABASE_URL}" ]; then
    DATABASE_URL=postgres://postgres:Password09@localhost:5006/server?sslmode=disable
fi

# App defaults for new devices / fresh clones
export APP_URL="${APP_URL:-http://localhost:8090}"
# Dashboard and server timezone (Asia/Jakarta = WIB, UTC+7)
export APP_TIMEZONE="${APP_TIMEZONE:-Asia/Jakarta}"
export TZ="${TZ:-$APP_TIMEZONE}"

# Set POSTGRES_URL from DATABASE_URL for the fondoger postgres fork
export POSTGRES_URL="${DATABASE_URL}"
# Use the database name from the URL for both data and aux (single database setup)
export POSTGRES_DATA_DB="server"
export POSTGRES_AUX_DB="server"

# Default serve arguments if none provided
if [ "$#" -eq 0 ]; then
    set -- serve --http=0.0.0.0:8090
fi

# In development mode, the Go backend proxies web/dashboard requests to the
# Vite dev server at localhost:5173. Start it here if needed.
SITE_DIR="$SCRIPT_DIR/internal/site"
if [ -f "$SITE_DIR/package.json" ]; then
    # Install frontend dependencies if not already installed
    if [ ! -d "$SITE_DIR/node_modules" ]; then
        echo "Installing frontend dependencies..."
        (cd "$SITE_DIR" && npm install --legacy-peer-deps)
    fi

    # Compile translations (.po -> .ts) required by vite dev server
    echo "Compiling translations..."
    (cd "$SITE_DIR" && npm run sync)

    # Start the Vite dev server in the background if it isn't already running
    if ! curl -s http://localhost:5173 >/dev/null 2>&1; then
        echo "Starting Vite dev server on localhost:5173..."
        (cd "$SITE_DIR" && npm run dev > /tmp/beszel-vite.log 2>&1 &)
        # Wait for Vite to become ready
        for i in $(seq 1 60); do
            if curl -s http://localhost:5173 >/dev/null 2>&1; then
                echo "Vite dev server is ready."
                break
            fi
            sleep 1
        done
    fi
fi

# Use dev build tag so migrations auto-generate and the dev web server proxy is used.
# For production, drop the development tag and build the production binary.
# Resolve Go from PATH so this script works on any device after cloning.
if command -v go >/dev/null 2>&1; then
    GO_BIN="go"
elif [ -x "/c/Program Files/Go/bin/go.exe" ]; then
    GO_BIN="/c/Program Files/Go/bin/go.exe"
else
    echo "Go not found. Install Go or set GO_BIN to the go executable." >&2
    exit 1
fi

exec "$GO_BIN" run -tags development ./internal/cmd/hub "$@"
