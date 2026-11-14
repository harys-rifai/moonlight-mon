package dbobs

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/sijms/go-ora/v2"
)

// CollectOracle gathers sessions, blocking, throughput, and slow SQL from Oracle.
func CollectOracle(ctx context.Context, target Target, samples *[]QuerySample) (Metrics, error) {
	var m Metrics
	service := target.Database
	if service == "" {
		service = "ORCL"
	}
	dsn := fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		target.Username, target.Password, target.Host, target.Port, service)

	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return m, err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	var banner string
	if err := db.QueryRowContext(ctx, "SELECT banner FROM v$version WHERE ROWNUM = 1").Scan(&banner); err != nil {
		return m, err
	}
	m.Up = true
	m.Version = firstWords(banner, 3)

	var sessions, blocked int
	var maxSessions float64
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM v$session WHERE type = 'USER'").Scan(&sessions)
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM v$session WHERE blocking_session IS NOT NULL").Scan(&blocked)
	_ = db.QueryRowContext(ctx, "SELECT TO_NUMBER(value) FROM v$parameter WHERE name = 'sessions'").Scan(&maxSessions)
	m.ConnsTotal = sessions
	m.ConnsMax = int(maxSessions)
	m.Blocked = blocked
	m.WaitingLocks = blocked
	if m.ConnsMax > 0 {
		m.ConnUsage = float64(m.ConnsTotal) / float64(m.ConnsMax) * 100
	}

	stat := func(name string) float64 {
		var value float64
		_ = db.QueryRowContext(ctx, "SELECT TO_NUMBER(value) FROM v$sysstat WHERE name = :1", name).Scan(&value)
		return value
	}
	m.QueriesTotal = int64(stat("execute count"))
	m.XactCommit = int64(stat("user commits"))
	m.XactRollback = int64(stat("user rollbacks"))
	m.Deadlocks = int64(stat("enqueue deadlocks"))

	logical := stat("db block gets") + stat("consistent gets")
	physical := stat("physical reads")
	if logical > 0 {
		m.CacheHitRatio = (1 - physical/logical) * 100
	}

	var sizeBytes int64
	if err := db.QueryRowContext(ctx, "SELECT COALESCE(SUM(bytes), 0) FROM dba_data_files").Scan(&sizeBytes); err != nil {
		_ = db.QueryRowContext(ctx, "SELECT COALESCE(SUM(bytes), 0) FROM user_segments").Scan(&sizeBytes)
	}
	m.DBSizeBytes = sizeBytes

	var slow, blocking int
	_ = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM v$session
		WHERE type = 'USER' AND status = 'ACTIVE' AND last_call_et > 60`).Scan(&slow)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM v$session WHERE blocking_session IS NOT NULL`).Scan(&blocking)
	m.SlowQueries = slow
	m.LongQueries = slow

	rows, err := db.QueryContext(ctx, `
		SELECT * FROM (
			SELECT sql_text, executions, elapsed_time / 1000, elapsed_time / 1000
			FROM v$sql
			ORDER BY elapsed_time DESC
		) WHERE ROWNUM <= 5`)
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

	m.Details = map[string]any{"blocked_sessions": blocking}
	return m, nil
}

func isOracleText(value string) bool {
	return strings.Contains(value, "Oracle")
}
