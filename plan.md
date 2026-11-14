# Moonlight Apps - Repair Plan (All Pages, Branding, PostgreSQL)

Date: 2026-11-14

## Scope
- Fix logo and rebrand "Beszel" -> "Moonlight apps" across all pages/UI surfaces
- Repair/verify PostgreSQL tables and indexes
- Document all missing/error items found and remediation status
- Repair run.sh and .env for new device deployment

## Issues Fixed

### 1. run.sh - DATABASE_URL unbound variable (Critical)
| Area | Issue | Status |
|---|---|---|
| `run.sh` | `DATABASE_URL: unbound variable` when `.env` doesn't exist | **Fixed** - Added default fallback (lines 24-26) |
| `run.sh` | npm install fails due to @lingui version conflict | **Fixed** - Added `--legacy-peer-deps` flag (line 46) |
| Missing `.env` on new devices | No DATABASE_URL configured | **Fixed** - `.env` created with default credentials |

### 2. .env file creation
- Created `.env` at project root with `DATABASE_URL=postgres://postgres:Password09@localhost:5006/server?sslmode=disable`
- Auto-generates on new devices without `.env`

### 3. Git identity
- Configured `user.email` and `user.name` for push.sh operations

## PostgreSQL Status
- Database: `postgres://postgres@localhost:5006/server` (connected OK)
- 29 tables, 69 indexes inventory verified
- Extensions: `pgcrypto`, `uuid_generate_v7`, `hex`, `randomblob`, `nocase` collation provisioned

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
- [x] `.env` file created with default credentials
- [x] `run.sh` DATABASE_URL default fallback added
- [x] `run.sh` npm install uses `--legacy-peer-deps`
- [x] `run.sh` Go runtime configured at `C:\Program Files\Go\`
- [ ] All pages visually verified in browser with cleared cache
- [ ] Locale files regenerated and reviewed
- [ ] Test suite updated/passing for renamed APIs
