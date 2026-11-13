# Errors and Resolutions (Server / Moonlight apps)

## 1. Frontend build failures (TypeScript, `npm run build` / `make build-hub`)

### 1.1 `Trans` not found in route pages
- **Files:** `internal/site/src/components/routes/databases.tsx`, `internal/site/src/components/routes/monitors.tsx`
- **Symptoms:** `tsc -b` errors `TS2304: Cannot find name 'Trans'` at JSX usages (`<Trans>Loading...</Trans>`, auth prompts/buttons).
- **Cause:** Components used `<Trans>` but imported only `useLingui` from `@lingui/react/macro`.
- **Resolution:** `import { useLingui, Trans } from "@lingui/react/macro"`.

### 1.2 `emptyMessage` handling in `NetworkMonitorsTableNew`
- **File:** `internal/site/src/components/network-monitors-table/network-monitors-table.tsx`
- **Symptoms:** `TS6133`/`TS2304` around optional `emptyMessage` during strict `tsc -b` builds.
- **Cause:** `NetworkMonitorsTableNew` renders rows via inner memoized `NetworkMonitorsTable`; `emptyMessage` was not threaded through, and stale `.tsbuildinfo` under `node_modules/.tmp` preserved bad incremental state.
- **Resolution:** Pass `emptyMessage` from `NetworkMonitorsTableNew` to inner `NetworkMonitorsTable` and render `emptyMessage ?? <Trans>No results.</Trans>` in the empty-state cell. `title` remains on public props and is used by the header. Clean before rebuild: `cd internal/site && npx tsc -b --clean && rm -rf node_modules/.tmp dist`.

### 1.3 `logo.tsx` JSX syntax error (broke lingui/Vite parsing)
- **File:** `internal/site/src/components/logo.tsx`
- **Symptoms:** Lingui/Babel parse error `Unexpected token, expected "}"` / `Unterminated JSX contents`; later `tsc` JSX tag errors.
- **Cause:** Malformed `<text>` attributes (`font-family`/`font-weight`/`font-size` in JSX instead of camelCase, and unclosed `fill={\`url(#${id})\`` missing closing brace/paren).
- **Resolution:** Use valid JSX SVG props and close expression: `fontFamily`, `fontWeight`, `fontSize`, `fill={\`url(#${id})\`}`.

### Verification
- `cd internal/site && npx tsc -b --clean && rm -rf node_modules/.tmp dist && npm run build`
- `cd /Users/harysrifai/web/Server && make build-hub`
- `make build-agent` → `build/beszel-agent_darwin_arm64` (previously succeeded).

## 2. Dev/runtime log noise observed (hub + Vite proxy)

- **`http: proxy error: context canceled`**, **`POST /api/realtime` → `unexpected EOF`**, **`Realtime connection closed (cancelled request|closed channel)`**
  - Typical during Vite HMR reloads, tab refresh/navigation, or client reconnect churn (browser aborts in-flight realtime/proxied requests). Benign if UI reconnects and APIs succeed. Investigate only if persistent: check `DATABASE_URL`/`POSTGRES_URL`, DB reachability, Vite proxy target `localhost:5173`, and timeouts.
- **High volume `INFO GET ...` for Vite modules/deps (`?t=`/`?v=`)** — normal dev/HMR fetching; reduced in production builds (non-development hub, built frontend assets).
- **Realtime DB/bridge activity** (`_realtimeClients/_realtimeChannels`, `pg_notify shared_bridge_channel`) — expected PocketBase/Postgres realtime subscription management.

### 1.4 Lingui macro / Vite build toolchain mismatch
- **Files:** `internal/site/vite.config.ts`, `internal/site/package.json` (dev deps)
- **Symptoms:** `failed to invoke ... @lingui/swc-plugin ... swc_plugin_runner` mismatch with `@vitejs/plugin-react-swc`; or `macro ... executed outside the context of compilation` when SWC macro plugin removed without alternative.
- **Cause:** Installed `@lingui/swc-plugin`/SWC host versions incompatible; Lingui macros still required transformation during Vite build.
- **Resolution:** Use `@vitejs/plugin-react` (Babel) with `babel-plugin-macros` in `vite.config.ts` alongside `@lingui/vite-plugin` and compiled catalogs. Keep `tsc -b` clean before build.

### 1.5 Footer repo/version link
- **File:** `internal/site/src/components/footer-repo-link.tsx`
- **Resolution:** GitHub/repo links point to `https://github.com/harys-rifai/moonlight-mon` (keeps “Moonlight apps {HUB_VERSION}”).

## 2. Dev/runtime log noise observed (hub + Vite proxy)

- **`http: proxy error: context canceled`**, **`POST /api/realtime` → `unexpected EOF`**, **`Realtime connection closed (cancelled request|closed channel)`** — typical during Vite HMR/reloads/reconnects; benign if UI/API recover. Investigate only if persistent (`DATABASE_URL`/`POSTGRES_URL`, DB reachability, Vite proxy `localhost:5173`, timeouts).
- **High volume `INFO GET ...` for Vite modules/deps (`?t=`/`?v=`)** — normal dev/HMR; reduced in production builds.
- **Realtime DB/bridge activity** (`_realtimeClients/_realtimeChannels`, `pg_notify shared_bridge_channel`) — expected.

## 3. Build environment notes

- `bun.lock` reported unknown lockfile version during `make build-hub`; bun warned and proceeded/regenerated as needed. Frontend build uses npm/bun-installed deps per Makefile.
- Lingui/SWC macro plugin incompatible with installed SWC host; resolved via Babel macros path (see §1.4).
- `@vitejs/plugin-react@6` peers with Vite 8; pinned compatible React plugin for Vite 7 in this workspace.
- Agent and hub builds complete on darwin/arm64 (see verification).

## 4. “No plan for redis.md / plan.md” in this repo

- No `*plan*redis*` or `plan.md` files exist under `/Users/harysrifai/web/Server`.
- Read for context: `internal/site/node_modules/bun-types/docs/api/redis.md` and external plans at `/Users/harysrifai/web/metric/plan.md`, `/Users/harysrifai/web/aerospike/plan.md`, `/Users/harysrifai/web/cursor/plan.md`, `/Users/harysrifai/web/Apotek/ApotekApps/howto/plan.md`.

## Verification (current)

- Frontend: `cd internal/site && npx tsc -b --clean && rm -rf node_modules/.tmp dist && npm run build` — **succeeded** (tsc clean, lingui extract/compile, Vite production build).
- Agent: `make build-agent` — **succeeded** (`build/beszel-agent_darwin_arm64`).
- Hub: `make build-hub` — **succeeded** (`build/beszel_darwin_arm64`).
- Combined: `make build` builds agent and hub (and web UI unless skipped).

## 5. Local database added to app (PostgreSQL on this machine)
- **Detected:** PostgreSQL 18 listening on `127.0.0.1:5006` (Homebrew, `postgres` PID present); `DATABASE_URL=postgres://postgres@localhost:5006/server?sslmode=disable`.
- **Action:** Inserted system and TCP network monitor in `server` DB so it appears in the app (Databases/Monitors filtered by DB ports incl. 5006):
  - `systems`: `id=syslocalpg`, `name=Local Machine`, `host=127.0.0.1`, `port=45876`, `status=down`, `users=["zht0w1qiv5f0i8y"]`
  - `network_monitors`: `id=2c3b6f3` (stable hash for syslocalpg/127.0.0.1/tcp/5006), `system=syslocalpg`, `target=127.0.0.1`, `port=5006`, `protocol=tcp`, `interval=60`, `enabled=true`
- **Notes:** Monitor response/status updates require a connected agent/system check cycle; UI will list the database monitor under `/databases` (and `/monitors`) when logged in. Other local DB names on port 5006: `apotek_pos`, `obs_db`, `postgres`, `server` (same port endpoint).
