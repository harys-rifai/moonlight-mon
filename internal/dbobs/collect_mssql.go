package dbobs

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

// CollectSQLServer gathers sessions, blocking, throughput, and slow statements.
func CollectSQLServer(ctx context.Context, target Target, samples *[]QuerySample) (Metrics, error) {
	var m Metrics
	dbname := target.Database
	if dbname == "" {
		dbname = "master"
	}
	query := url.Values{}
	query.Set("database", dbname)
	query.Set("connection timeout", "5")
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%d?%s", target.Username, target.Password, target.Host, target.Port, query.Encode())

	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return m, err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var version string
	if err := db.QueryRowContext(ctx, "SELECT @@VERSION").Scan(&version); err != nil {
		return m, err
	}
	m.Up = true
	m.Version = "SQL Server " + firstWords(strings.Split(version, "\n")[0], 2)
	m.ConnsMax = 32767

	var sessions, running, blocked int
	_ = db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM sys.dm_exec_sessions WHERE is_user_process = 1),
			(SELECT COUNT(*) FROM sys.dm_exec_requests WHERE session_id <> @@SPID),
			(SELECT COUNT(*) FROM sys.dm_exec_requests WHERE blocking_session_id <> 0)`).Scan(&sessions, &running, &blocked)
	m.ConnsTotal = sessions
	m.ConnsActive = running
	m.ConnsIdle = sessions - running
	m.Blocked = blocked
	m.WaitingLocks = blocked
	if m.ConnsMax > 0 {
		m.ConnUsage = float64(m.ConnsTotal) / float64(m.ConnsMax) * 100
	}

	var queriesTotal int64
	_ = db.QueryRowContext(ctx, "SELECT COALESCE(SUM(execution_count), 0) FROM sys.dm_exec_query_stats").Scan(&queriesTotal)
	m.QueriesTotal = queriesTotal

	var sizeBytes int64
	_ = db.QueryRowContext(ctx, "SELECT COALESCE(SUM(CAST(size AS bigint)) * 8192, 0) FROM sys.database_files").Scan(&sizeBytes)
	m.DBSizeBytes = sizeBytes

	var slowCount int
	_ = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sys.dm_exec_requests
		WHERE total_elapsed_time > 60000 AND session_id <> @@SPID`).Scan(&slowCount)
	m.SlowQueries = slowCount
	m.LongQueries = slowCount

	var waits float64
	_ = db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(wait_time), 0) FROM sys.dm_exec_requests
		WHERE blocking_session_id <> 0`).Scan(&waits)
	m.LockWaitSec = waits / 1000

	rows, err := db.QueryContext(ctx, `
		SELECT TOP 5 t.text, s.execution_count,
		       s.total_elapsed_time / NULLIF(s.execution_count, 0) / 1000.0,
		       s.total_elapsed_time / 1000.0
		FROM sys.dm_exec_query_stats s
		CROSS APPLY sys.dm_exec_sql_text(s.sql_handle) t
		ORDER BY s.total_elapsed_time DESC`)
	if err == nil {
		for rows.Next() {
			var text string
			var calls int64
			var meanMs, totalMs float64
			if err := rows.Scan(&text, &calls, &meanMs, &totalMs); err != nil {
				break
			}
			*samples = append(*samples, QuerySample{Query: text, Calls: calls, MeanMs: meanMs, TotalMs: totalMs})
		}
		rows.Close()
	}

	m.Details = map[string]any{"blocked_sessions": blocked}
	return m, nil
}

func firstWords(value string, count int) string {
	fields := strings.Fields(value)
	if len(fields) > count {
		fields = fields[:count]
	}
	return strings.Join(fields, " ")
}
