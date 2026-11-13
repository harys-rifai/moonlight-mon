# Errors and Resolutions (Server / Moonlight apps)

## 1. Frontend build failures (TypeScript, `npm run build` / `make build-hub`)

### 1.1 `Trans` not found in route pages
- **Files:** `internal/site/src/components/routes/databases.tsx`, `internal/site/src/components/routes/monitors.tsx`
- **Symptoms:** `tsc -b` errors `TS2304: Cannot find name 'Trans'` at JSX usages (`<Trans>Loading...</Trans>`, auth prompts/buttons).
- **Cause:** Components used `<Trans>` but imported only `useLingui` from `@lingui/react/macro`.
- **Resolution:** `import { useLingui, Trans } from "@lingui/react/macro"`.

### 1.2 `emptyMessage` / unused-prop handling in `NetworkMonitorsTableNew`
- **File:** `internal/site/src/components/network-monitors-table/network-monitors-table.tsx`
- **Symptoms:** `TS6133` unused parameter and/or `TS2304` cannot-find-name during strict `tsc -b` builds for the optional `emptyMessage` prop in the empty-state cell.
- **Cause:** Strict unused checks on an optional prop used in a conditional render path interacted poorly with incremental/build-info state during repeated edits.
- **Resolution:** Read the prop into a local and render it in the empty state: `const resolvedEmptyMessage = emptyMessage` and `resolvedEmptyMessage ?? <Trans>No results.</Trans>`. Keep `title` accepted (callers pass it); if unused in header, mark intentionally (`void title`) or render it. After edits, clean TS build artifacts before rebuilding: `cd internal/site && npx tsc -b --clean && rm -rf node_modules/.tmp dist`.

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

## 3. Build environment notes

- `bun.lock` reported unknown lockfile version during `make build-hub`; bun warned and proceeded/regenerated as needed. Frontend build uses npm/bun-installed deps per Makefile.
- `@lingui/swc-plugin`/SWC compatibility errors can occur if plugin and host SWC versions mismatch; if seen, align `@lingui/swc-plugin`, `@lingui/*`, and `@vitejs/plugin-react-swc` versions or pin compatible releases.
- `make build-agent` completed successfully on darwin/arm64.

## 4. “No plan for redis.md / plan.md” in this repo

- No `*plan*redis*` or `plan.md` files exist under `/Users/harysrifai/web/Server`.
- Read for context: `internal/site/node_modules/bun-types/docs/api/redis.md` and external plans at `/Users/harysrifai/web/metric/plan.md`, `/Users/harysrifai/web/aerospike/plan.md`, `/Users/harysrifai/web/cursor/plan.md`, `/Users/harysrifai/web/Apotek/ApotekApps/howto/plan.md`.
