# Moonlight Apps - Repair Plan (All Pages, Branding, PostgreSQL)

Date: 2026-11-13

## Scope
- Fix logo and rebrand "Beszel" -> "Moonlight apps" across all pages/UI surfaces
- Repair/verify PostgreSQL tables and indexes
- Document all missing/error items found and remediation status

## Issues Found

### 1. Branding / Logo / UI (Critical)
| Area | Issue | Status |
|---|---|---|
| `internal/site/src/components/logo.tsx` | SVG path rendered old "Beszel" wordmark glyph | **Fixed** (replaced with "Moonlight" SVG text + gradient) |
| `internal/site/index.html` | `globalThis.BESZEL` injected | **Fixed** -> `globalThis.MOONLIGHT` |
| `internal/site/src/types.d.ts` | `BESZEL` global type and `BeszelInfo` | **Fixed** -> `MOONLIGHT`, `MoonlightInfo` |
| `internal/site/src/main.tsx` | API calls to `/api/beszel/*`, missing `databases` route render | **Fixed** (moonlight APIs, added Databases route) |
| Router/base path consumers | `globalThis.BESZEL` reads | **Fixed** (`router.tsx`, `utils.ts`, login, auth, footer, systems table) |
| Frontend API consumers | `/api/beszel/*` endpoints | **Fixed** to `/api/moonlight/*` in all source components |
| Backend API routes | `/api/beszel` route group | **Fixed** to `/api/moonlight` in `internal/hub/api.go` |
| SPA route allowlist | `/databases` missing in `isAppRoute` | **Fixed** in `internal/hub/server.go` |
| Manifest/package | `Beszel`/`beszel` names | **Fixed** (`manifest.json`, `package.json`) |
| Email templates | Brand text "Beszel" | **Fixed** (`layout.html`, `alert.html`) |
| Install commands UI | Docker container/volume names used `beszel-*` | **Partially fixed** (user-facing names -> `moonlight-*`; image/script URLs still upstream) |
| Footer/docs links | Links to `beszel.dev` / `henrygd/beszel` repo | **Pending** (external upstream URLs; change if Moonlight has own docs/repo) |
| Locale `.po`/compiled `.ts` | Many historical "Beszel" strings and old API references | **Pending** (requires Lingui extract/compile and translation review) |
| Tests | Many tests expect `/api/beszel/*`, `BESZEL_*` env vars, module names | **Pending** (large refactor; update alongside API/env compatibility strategy) |
| Deployment artifacts | Dockerfiles, Helm charts, scripts, goreleaser, supplemental docs reference Beszel | **Pending** (separate release/rebrand task) |
| Core Go module/package | Module path `github.com/henrygd/beszel`, package `beszel`, binary names | **Pending/intentional** (kept for compatibility; `AppName` set to `moonlight`) |

### 2. Blank "Network Apps / Databases" Page
| Cause | Fix | Status |
|---|---|---|
| `databases` route existed in router/navbar but not rendered in `App` switch (`main.tsx`) | Added lazy import and route case for `Databases` | **Fixed** |
| Production SPA route check omitted `/databases` | Added `/databases` to `isAppRoute` allowlist | **Fixed** |
| Frontend still using old global/API names in places | Migrated to `MOONLIGHT` + `/api/moonlight/*` | **Fixed** |
| Data visibility depends on auth and monitor records/ports | Verify logged-in user and `network_monitors` records exist | **Verify** |

### 3. PostgreSQL Tables & Indexes
**Database checked:** `postgres://postgres@localhost:5006/server` (connected OK)

**Tables present (29):** `_authOrigins`, `_collections`, `_externalAuths`, `_logs`, `_mfas`, `_migrations`, `_otps`, `_params`, `_realtimeChannels`, `_realtimeClients`, `_superusers`, `alerts`, `alerts_history`, `container_stats`, `containers`, `fingerprints`, `network_monitor_stats`, `network_monitors`, `quiet_hours`, `smart_devices`, `system_details`, `system_stats`, `systemd_services`, `systems`, `test_collection`, `universal_tokens`, `user_settings`, `users`, `zfs_pools`

**Indexes present (69):** All core primary keys and expected app indexes found, including:
- `network_monitor_stats`: `idx_nms_monitor_type_created`, `idx_nms_system_type_created`, `idx_nms_type_created`
- `network_monitors`: `idx_nm_system_enabled`
- `alerts`: `idx_MnhEt21L5r` (unique user/system/name), `idx_alerts_system_name`
- `system_stats`: `idx_GxIee0j (system,type,created)`
- `container_stats`: `idx_d87OiXGZD8 (system,type,created)`
- `users`: email/tokenKey/username unique indexes
- `_collections`: unique name index

**Assessment:** No missing core tables or indexes detected in current schema snapshot. PostgreSQL extensions/functions required by migrations (`pgcrypto`, `uuid_generate_v7`, `hex`, `randomblob`, `nocase` collation) are provisioned by app bootstrap/migrations.

**Repair actions (if drift occurs):**
1. Backup DB before changes
2. Run hub with dev/migration flow to reapply pending migrations: `./run.sh`
3. Rebuild missing indexes idempotently (example):
   - `CREATE INDEX IF NOT EXISTS idx_nms_monitor_type_created ON network_monitor_stats (monitor,type,created);`
   - `CREATE INDEX IF NOT EXISTS idx_nms_system_type_created ON network_monitor_stats (system,type,created);`
   - `CREATE INDEX IF NOT EXISTS idx_nm_system_enabled ON network_monitors (system,enabled);`
4. Validate constraints/index usage with `EXPLAIN` and `pg_stat_user_indexes`

### 4. Runtime / Environment
| Item | Finding | Status |
|---|---|---|
| PostgreSQL | Running on `localhost:5006`, DB `server` accessible | **OK** |
| Redis | `redis-cli 8.6.2` installed; no app integration yet | **Not integrated** (see `planredis.md`) |
| Backend | Hub running (`/api/health` 200, `/api/moonlight/first-run` OK) | **OK** |
| Frontend dev server | Vite running on 5173 | **OK** |
| API compatibility | Backend moved to `/api/moonlight/*`; old `/api/beszel/*` removed from router group | **Breaking for old clients/tests** |

## Pending / Follow-up Work
1. Run Lingui extract/compile and update translations to remove remaining "Beszel" strings in locales.
2. Decide and implement external rebrand targets (docs URL, GitHub repo, Docker image names, install scripts) or keep upstream-compatible fallbacks.
3. Update Go/TS tests for new API prefix (`/api/moonlight/*`) and any renamed env/global identifiers if a full rename is desired.
4. Add compatibility aliases (temporary): serve `/api/beszel/*` -> `/api/moonlight/*` and accept both `BESZEL_HUB_*`/`MOONLIGHT_HUB_*` env prefixes during migration window.
5. Smoke-test all pages in browser after hard refresh (Safari cache issue noted): Home, Containers, S.M.A.R.T., Monitors, Databases, System detail, Settings tabs, Login/Auth flows.

## Verification Checklist
- [x] Logo component no longer uses Beszel SVG glyph path
- [x] `index.html` injects `globalThis.MOONLIGHT`
- [x] Frontend source uses `/api/moonlight/*` endpoints
- [x] Backend serves `/api/moonlight/*` routes
- [x] `/databases` route renders in SPA and allowed by server route check
- [x] PostgreSQL tables/indexes inventoried (29 tables, 69 indexes)
- [ ] All pages visually verified in browser with cleared cache
- [ ] Locale files regenerated and reviewed
- [ ] Test suite updated/passing for renamed APIs
