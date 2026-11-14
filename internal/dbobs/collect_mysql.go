package dbobs

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// CollectMySQL gathers health and performance metrics from MySQL or MariaDB.
func CollectMySQL(ctx context.Context, target Target, samples *[]QuerySample) (Metrics, error) {
	var m Metrics
	user := target.Username
	password := target.Password
	dbname := target.Database
	if dbname == "" {
		dbname = "mysql"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=5s&readTimeout=10s&parseTime=true", user, password, target.Host, target.Port, dbname)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return m, err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var version string
	var maxConns int
	if err := db.QueryRowContext(ctx, "SELECT VERSION(), @@max_connections").Scan(&version, &maxConns); err != nil {
		return m, err
	}
	m.Up = true
	m.Version = version
	m.ConnsMax = maxConns

	status := map[string]float64{}
	rows, err := db.QueryContext(ctx, "SHOW GLOBAL STATUS")
	if err == nil {
		for rows.Next() {
			var name, value string
			if err := rows.Scan(&name, &value); err != nil {
				break
			}
			if parsed, err := strconv.ParseFloat(value, 64); err == nil {
				status[name] = parsed
			}
		}
		rows.Close()
	}

	m.UptimeSec = status["Uptime"]
	m.ConnsTotal = int(status["Threads_connected"])
	m.ConnsActive = int(status["Threads_running"])
	m.ConnsIdle = m.ConnsTotal - m.ConnsActive
	if m.ConnsMax > 0 {
		m.ConnUsage = float64(m.ConnsTotal) / float64(m.ConnsMax) * 100
	}
	m.XactCommit = int64(status["Com_commit"])
	m.XactRollback = int64(status["Com_rollback"])
	m.QueriesTotal = int64(status["Questions"])
	m.Blocked = int(status["Innodb_row_lock_current_waits"])
	m.Deadlocks = int64(status["Innodb_deadlocks"])
	m.SlowQueries = int(status["Slow_queries"])
	m.IdleInTx = 0
	m.WaitingLocks = m.Blocked
	m.TempBytes = int64(status["Created_tmp_disk_tables"])

	reads := status["Innodb_buffer_pool_reads"]
	readRequests := status["Innodb_buffer_pool_read_requests"]
	if readRequests > 0 {
		m.CacheHitRatio = (1 - reads/readRequests) * 100
	}

	// Size and long-running statements.
	if err := db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(data_length + index_length), 0) FROM information_schema.tables WHERE table_schema = DATABASE()",
	).Scan(&m.DBSizeBytes); err != nil {
		m.DBSizeBytes = 0
	}

	slowRows, err := db.QueryContext(ctx, `
		SELECT ID, USER, TIME, STATE, INFO
		FROM information_schema.processlist
		WHERE COMMAND = 'Query' AND TIME > 5 AND INFO IS NOT NULL
		ORDER BY TIME DESC
		LIMIT 5`)
	if err == nil {
		for slowRows.Next() {
			var id, slowUser, state sql.NullString
			var seconds sql.NullInt64
			var info sql.NullString
			if err := slowRows.Scan(&id, &slowUser, &seconds, &state, &info); err != nil {
				break
			}
			secs := float64(seconds.Int64)
			label := info.String
			if state.String != "" {
				label = fmt.Sprintf("-- state=%s\n%s", state.String, label)
			}
			*samples = append(*samples, QuerySample{
				Query:   fmt.Sprintf("%s: %s", slowUser.String, label),
				Calls:   1,
				MeanMs:  secs * 1000,
				TotalMs: secs * 1000,
			})
		}
		slowRows.Close()
	}

	m.LongQueries = int(status["Slow_queries"])
	m.Details = map[string]any{
		"aborted_connects":    status["Aborted_connects"],
		"tmp_disk_tables":     status["Created_tmp_disk_tables"],
		"table_locks_waited":  status["Table_locks_waited"],
		"threads_running":     status["Threads_running"],
		"queries_per_second":  status["Queries"],
		"open_tables":         status["Open_tables"],
	}
	return m, nil
}

// mysqlVersionString normalizes MariaDB/MySQL version prefixes.
func mysqlVersionString(version string) string {
	if strings.Contains(strings.ToLower(version), "mariadb") {
		return "MariaDB " + strings.Fields(version)[0]
	}
	return "MySQL " + strings.Fields(version)[0]
}
