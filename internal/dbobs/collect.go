// Package dbobs implements the database observability module: collectors, metric
// storage, threshold evaluation, alert lifecycle, and notifications.
package dbobs

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// QuerySample is one entry from pg_stat_statements (APM view).
type QuerySample struct {
	Query     string
	Calls     int64
	MeanMs    float64
	TotalMs   float64
	Rows      int64
	CacheHit  float64
	Statement string
}

// Metrics is one collection sample for a database target.
type Metrics struct {
	Up             bool
	Version        string
	UptimeSec      float64
	ConnsTotal     int
	ConnsActive    int
	ConnsIdle      int
	ConnsMax       int
	ConnUsage      float64
	LongQueries    int
	Blocked        int
	LockWaitSec    float64
	ReplicationLag float64
	DBSizeBytes    int64
	OldestTxSec    float64

	// Deep PostgreSQL metrics. Counters are cumulative and converted to rates by the runner.
	XactCommit    int64
	XactRollback  int64
	QueriesTotal  int64
	BlksRead      int64
	BlksHit       int64
	TempBytes     int64
	WalBytes      int64
	Deadlocks     int64
	SlowQueries   int
	IdleInTx      int
	WaitingLocks  int
	BlkReadTimeMs float64
	BlkWriteTime  float64
	CacheHitRatio float64
}

// DSN builds a PostgreSQL connection string for an inventory record.
func DSN(host string, port int, dbname, user, password, sslmode string) string {
	if dbname == "" {
		dbname = "postgres"
	}
	if sslmode == "" {
		sslmode = "disable"
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&connect_timeout=5",
		urlEscape(user), urlEscape(password), host, port, dbname, sslmode,
	)
}

func urlEscape(value string) string {
	replacer := strings.NewReplacer(
		"@", "%40", "/", "%2F", ":", "%3A", "?", "%3F", "#", "%23", "&", "%26",
	)
	return replacer.Replace(value)
}

// CollectPostgres gathers availability, capacity, performance, locking, and APM
// metrics. A failed connection returns Metrics{Up:false} plus the error.
func CollectPostgres(ctx context.Context, dsn string) (Metrics, []QuerySample, error) {
	var m Metrics
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return m, nil, err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	const query = `
		SELECT
			coalesce((SELECT count(*) FROM pg_stat_activity WHERE backend_type = 'client backend'), 0),
			coalesce((SELECT count(*) FROM pg_stat_activity WHERE backend_type = 'client backend' AND state = 'active'), 0),
			coalesce((SELECT count(*) FROM pg_stat_activity WHERE backend_type = 'client backend' AND state = 'idle'), 0),
			coalesce(current_setting('max_connections')::int, 0),
			coalesce((SELECT count(*) FROM pg_stat_activity WHERE state = 'active' AND now() - query_start > interval '5 minutes'), 0),
			coalesce((SELECT count(*) FROM pg_stat_activity WHERE wait_event_type = 'Lock'), 0),
			coalesce((SELECT max(extract(epoch FROM now() - state_change)) FROM pg_stat_activity WHERE wait_event_type = 'Lock'), 0),
			coalesce((SELECT max(extract(epoch FROM replay_lag)) FROM pg_stat_replication), 0),
			coalesce(pg_database_size(current_database()), 0),
			coalesce((SELECT max(extract(epoch FROM now() - xact_start)) FROM pg_stat_activity WHERE xact_start IS NOT NULL), 0),
			coalesce(extract(epoch FROM now() - pg_postmaster_start_time()), 0),
			version(),
			coalesce(d.xact_commit, 0),
			coalesce(d.xact_rollback, 0),
			coalesce(d.blks_read, 0),
			coalesce(d.blks_hit, 0),
			coalesce(d.temp_bytes, 0),
			coalesce(d.deadlocks, 0),
			coalesce(d.blk_read_time, 0),
			coalesce(d.blk_write_time, 0),
			coalesce(w.wal_bytes, 0),
			coalesce((SELECT count(*) FROM pg_stat_activity WHERE state = 'idle in transaction'), 0),
			coalesce((SELECT count(*) FROM pg_stat_activity WHERE state = 'active' AND now() - query_start > interval '1 minute'), 0),
			coalesce((SELECT sum(calls) FROM pg_stat_statements), 0)
		FROM pg_stat_database d
		CROSS JOIN (SELECT wal_bytes FROM pg_stat_wal LIMIT 1) w
		WHERE d.datname = current_database()`

	// pg_stat_statements may be unavailable; fall back to a version without APM counters.
	err = db.QueryRowContext(ctx, query).Scan(
		&m.ConnsTotal, &m.ConnsActive, &m.ConnsIdle, &m.ConnsMax,
		&m.LongQueries, &m.Blocked, &m.LockWaitSec, &m.ReplicationLag,
		&m.DBSizeBytes, &m.OldestTxSec, &m.UptimeSec, &m.Version,
		&m.XactCommit, &m.XactRollback, &m.BlksRead, &m.BlksHit,
		&m.TempBytes, &m.Deadlocks, &m.BlkReadTimeMs, &m.BlkWriteTime,
		&m.WalBytes, &m.IdleInTx, &m.SlowQueries, &m.QueriesTotal,
	)
	var samples []QuerySample
	if err != nil {
		const fallback = `
			SELECT
				coalesce((SELECT count(*) FROM pg_stat_activity WHERE backend_type = 'client backend'), 0),
				coalesce((SELECT count(*) FROM pg_stat_activity WHERE backend_type = 'client backend' AND state = 'active'), 0),
				coalesce((SELECT count(*) FROM pg_stat_activity WHERE backend_type = 'client backend' AND state = 'idle'), 0),
				coalesce(current_setting('max_connections')::int, 0),
				coalesce((SELECT count(*) FROM pg_stat_activity WHERE state = 'active' AND now() - query_start > interval '5 minutes'), 0),
				coalesce((SELECT count(*) FROM pg_stat_activity WHERE wait_event_type = 'Lock'), 0),
				coalesce((SELECT max(extract(epoch FROM now() - state_change)) FROM pg_stat_activity WHERE wait_event_type = 'Lock'), 0),
				coalesce((SELECT max(extract(epoch FROM replay_lag)) FROM pg_stat_replication), 0),
				coalesce(pg_database_size(current_database()), 0),
				coalesce((SELECT max(extract(epoch FROM now() - xact_start)) FROM pg_stat_activity WHERE xact_start IS NOT NULL), 0),
				coalesce(extract(epoch FROM now() - pg_postmaster_start_time()), 0),
				version(),
				coalesce(d.xact_commit, 0),
				coalesce(d.xact_rollback, 0),
				coalesce(d.blks_read, 0),
				coalesce(d.blks_hit, 0),
				coalesce(d.temp_bytes, 0),
				coalesce(d.deadlocks, 0),
				coalesce(d.blk_read_time, 0),
				coalesce(d.blk_write_time, 0),
				coalesce((SELECT count(*) FROM pg_stat_activity WHERE state = 'idle in transaction'), 0),
				coalesce((SELECT count(*) FROM pg_stat_activity WHERE state = 'active' AND now() - query_start > interval '1 minute'), 0)
			FROM pg_stat_database d
			WHERE d.datname = current_database()`
		if err := db.QueryRowContext(ctx, fallback).Scan(
			&m.ConnsTotal, &m.ConnsActive, &m.ConnsIdle, &m.ConnsMax,
			&m.LongQueries, &m.Blocked, &m.LockWaitSec, &m.ReplicationLag,
			&m.DBSizeBytes, &m.OldestTxSec, &m.UptimeSec, &m.Version,
			&m.XactCommit, &m.XactRollback, &m.BlksRead, &m.BlksHit,
			&m.TempBytes, &m.Deadlocks, &m.BlkReadTimeMs, &m.BlkWriteTime,
			&m.IdleInTx, &m.SlowQueries,
		); err != nil {
			return m, nil, err
		}
	}

	m.Up = true
	m.Version = strings.TrimSpace(m.Version)
	if i := strings.Index(m.Version, " on "); i > 0 {
		m.Version = m.Version[:i]
	}
	if m.ConnsMax > 0 {
		m.ConnUsage = float64(m.ConnsTotal) / float64(m.ConnsMax) * 100
	}
	if m.BlksHit+m.BlksRead > 0 {
		m.CacheHitRatio = float64(m.BlksHit) / float64(m.BlksHit+m.BlksRead) * 100
	}
	m.WaitingLocks = m.Blocked

	samples = collectStatementSamples(ctx, db)
	if len(samples) == 0 {
		// Without pg_stat_statements, still capture live slow statements for APM detail.
		samples = collectLiveSlowQueries(ctx, db)
	}
	return m, samples, nil
}

// collectLiveSlowQueries captures currently running statements longer than 5 seconds.
func collectLiveSlowQueries(ctx context.Context, db *sql.DB) []QuerySample {
	const q = `
		SELECT usename, query, extract(epoch FROM now() - query_start) * 1000,
		       coalesce(wait_event_type, '')
		FROM pg_stat_activity
		WHERE state = 'active'
		  AND backend_type = 'client backend'
		  AND pid <> pg_backend_pid()
		  AND query NOT LIKE 'START_REPLICATION%'
		  AND now() - query_start > interval '5 seconds'
		ORDER BY 3 DESC
		LIMIT 5`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var samples []QuerySample
	for rows.Next() {
		var user, query, waitType string
		var durationMs float64
		if err := rows.Scan(&user, &query, &durationMs, &waitType); err != nil {
			return samples
		}
		label := query
		if waitType != "" {
			label = fmt.Sprintf("-- waiting on %s\n%s", waitType, query)
		}
		samples = append(samples, QuerySample{
			Query:   fmt.Sprintf("%s: %s", user, label),
			Calls:   1,
			MeanMs:  durationMs,
			TotalMs: durationMs,
		})
	}
	return samples
}

// collectStatementSamples reads the slowest statements when pg_stat_statements is installed.
func collectStatementSamples(ctx context.Context, db *sql.DB) []QuerySample {
	const q = `
		SELECT query, calls, mean_exec_time, total_exec_time, rows,
		       CASE WHEN shared_blks_hit + shared_blks_read > 0
		            THEN shared_blks_hit::float8 / (shared_blks_hit + shared_blks_read) * 100
		            ELSE 0 END
		FROM pg_stat_statements
		WHERE calls > 0
		ORDER BY total_exec_time DESC
		LIMIT 5`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var samples []QuerySample
	for rows.Next() {
		var sample QuerySample
		if err := rows.Scan(&sample.Query, &sample.Calls, &sample.MeanMs, &sample.TotalMs, &sample.Rows, &sample.CacheHit); err != nil {
			return samples
		}
		samples = append(samples, sample)
	}
	return samples
}
