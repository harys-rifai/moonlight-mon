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

// CollectPostgres gathers availability, capacity, and performance metrics.
// A failed connection returns Metrics{Up:false} plus the error for the record log.
func CollectPostgres(ctx context.Context, dsn string) (Metrics, error) {
	var m Metrics
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return m, err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
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
			version()`

	if err := db.QueryRowContext(ctx, query).Scan(
		&m.ConnsTotal, &m.ConnsActive, &m.ConnsIdle, &m.ConnsMax,
		&m.LongQueries, &m.Blocked, &m.LockWaitSec, &m.ReplicationLag,
		&m.DBSizeBytes, &m.OldestTxSec, &m.UptimeSec, &m.Version,
	); err != nil {
		return m, err
	}

	m.Up = true
	m.Version = strings.TrimSpace(m.Version)
	if i := strings.Index(m.Version, " on "); i > 0 {
		m.Version = m.Version[:i]
	}
	if m.ConnsMax > 0 {
		m.ConnUsage = float64(m.ConnsTotal) / float64(m.ConnsMax) * 100
	}
	return m, nil
}
