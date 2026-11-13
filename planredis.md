# Moonlight Apps - Redis Integration Plan (Seamless Database Sync)

Date: 2026-11-13
Goal: Add Redis to improve real-time sync, cache hot reads, and coordinate multi-instance hub deployments while PostgreSQL remains the source of truth.

## Current State
- PostgreSQL is the primary datastore (verified running on `localhost:5006`, DB `server`)
- `redis-cli 8.6.2` is available on host; no Redis client/config/integration in app code
- Hub uses PocketBase realtime mechanisms; multi-instance sync may be limited without a shared pub/sub/cache layer
- Frontend polls/subscribes via existing APIs; no Redis-aware behavior required initially

## Design Principles
- PostgreSQL = source of truth; Redis = cache/pub-sub/ephemeral coordination only
- All Redis features must be **optional** and degrade gracefully when Redis is unavailable
- Use namespaced keys with TTLs; avoid storing secrets in plaintext values
- Idempotent consumers and safe reconnect/backoff for pub-sub

## Proposed Architecture
1. **Connection & Config**
   - Add `REDIS_URL` (e.g. `redis://localhost:6379/0`) and optional `REDIS_PASSWORD`, `REDIS_TLS`, `REDIS_PREFIX`
   - Initialize Redis client at hub startup; health-check on boot and periodic ping
   - Expose Redis status in logs/health diagnostics (not public by default)

2. **Cache Layer (Read Optimization)**
   - Cache hot, frequently-read payloads: dashboard summaries, system lists, monitor stats snapshots, alert counts
   - Key pattern: `moonlight:{env}:cache:{resource}:{scope}:{version}`
   - TTLs: 5-30s for volatile metrics, 1-5m for slow-changing metadata
   - Invalidate/update on writes (system upserts, monitor changes, alert state changes)

3. **Pub/Sub for Seamless Sync**
   - Channel pattern: `moonlight:{env}:events:{topic}` (e.g. `systems`, `monitors`, `alerts`, `stats`)
   - Publish compact events on DB mutations: `{type, entity, id, action, ts}`
   - Subscribers refresh local caches and forward realtime updates to connected clients
   - Enables multi-hub instance consistency without sticky-session dependence

4. **Distributed Coordination (Multi-instance)**
   - Use Redis locks for singleton jobs (rollups, cleanup, heartbeat checks) to prevent duplicate work
   - Rate-limit and dedupe keys for noisy updates/notifications
   - Optional leader election key with short TTL + renewal

5. **Session/Transient State (Optional)**
   - Store ephemeral state (pending requests, short-lived tokens metadata, UI presence) only if needed
   - Never replace PocketBase auth/session storage without explicit migration

## Implementation Steps
1. **Dependencies & Config**
   - Add Go Redis client (e.g. `github.com/redis/go-redis/v9`)
   - Extend config/env parsing in `internal/cmd/hub` / hub config utilities
   - Add `.env` example entries and startup validation/logging

2. **Redis Package**
   - Create `internal/cache/redis` (or similar): client wrapper, connect/ping, get/set/del, pub/sub helpers, lock helpers
   - Centralize key builder and TTL constants

3. **Integrate Cache**
   - Wrap selected read paths with cache-aside (read cache -> miss -> DB -> set cache)
   - Add invalidation hooks on create/update/delete paths for systems, monitors, alerts, settings

4. **Integrate Pub/Sub**
   - Emit events after successful DB commits
   - Subscribe on startup; route events to existing realtime/broadcast/update mechanisms
   - Handle reconnect with exponential backoff and subscription re-establishment

5. **Operational Hardening**
   - Metrics/logs for hit rate, latency, reconnects, publish/consume errors
   - Configurable enable/disable per feature (`REDIS_CACHE_ENABLED`, `REDIS_PUBSUB_ENABLED`)
   - Timeouts and circuit-breaker behavior to protect DB/request latency

6. **Testing & Rollout**
   - Unit tests for key building, serialization, cache invalidation logic
   - Integration tests with Redis container/service (docker-compose) for pub/sub and locks
   - Staged rollout: dev -> staging -> prod; monitor hit rates and error rates

## PostgreSQL Considerations
- Keep all authoritative writes in PostgreSQL transactions
- Publish Redis events only after commit to avoid phantom updates
- Ensure indexes support cache-miss query paths (current indexes reviewed; add as needed based on query plans)
- For cross-instance consistency, prefer Redis pub/sub notifications + short TTL caches over long-lived stale caches

## Risks & Mitigations
| Risk | Mitigation |
|---|---|
| Cache staleness | Short TTLs, explicit invalidation on writes, event-driven refresh |
| Redis outage | Graceful fallback to DB-only mode, timeouts, health alerts |
| Message loss/duplication | Idempotent handlers, periodic reconciliation, sequence/timestamp checks |
| Key bloat/memory growth | Namespacing, TTLs everywhere, eviction policy (`allkeys-lru`/`volatile-ttl`) |
| Security exposure | Auth/TLS for Redis, network isolation, no sensitive payloads in keys/channels |

## Suggested .env Additions
```env
# Redis (optional)
REDIS_URL=redis://localhost:6379/0
REDIS_PREFIX=moonlight:dev
REDIS_CACHE_ENABLED=true
REDIS_PUBSUB_ENABLED=true
REDIS_TLS=false
```

## Acceptance Criteria
- App boots and operates normally with Redis disabled
- With Redis enabled: cache hits observed for selected endpoints, pub/sub propagates updates across instances
- No data inconsistency vs PostgreSQL after reconnect/restart scenarios
- Monitoring/logging confirms Redis health and error handling paths
