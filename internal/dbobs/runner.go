package dbobs

import (
	"context"
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Notifier delivers an alert message to the configured channels.
type Notifier func(title, message string)

// DefaultRules mirrors the thresholds from the observability design.
var DefaultRules = []struct {
	Metric   string
	Warning  float64
	Critical float64
}{
	{"down", 1, 1},
	{"connection_usage", 80, 90},
	{"long_queries", 1, 5},
	{"blocked_sessions", 10, 25},
	{"lock_wait_sec", 60, 120},
	{"replication_lag_sec", 60, 300},
	{"oldest_tx_sec", 300, 900},
}

// Start runs the collector loop until the app stops.
func Start(app core.App, notify Notifier) {
	interval := 60 * time.Second
	runOnce(app, notify)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		runOnce(app, notify)
	}
}

func runOnce(app core.App, notify Notifier) {
	inventory, err := app.FindAllRecords("db_inventory")
	if err != nil {
		app.Logger().Error("dbobs: failed to load inventory", "err", err)
		return
	}
	if len(inventory) == 0 {
		return
	}
	rules := loadRules(app)

	for _, item := range inventory {
		if !item.GetBool("enabled") {
			continue
		}
		if engine := item.GetString("engine"); engine != "" && engine != "postgres" {
			// Only PostgreSQL has a collector in this build; others are inventory-only.
			continue
		}
		sampleDatabase(app, item, rules, notify)
	}
}

func sampleDatabase(app core.App, item *core.Record, rules map[string]rule, notify Notifier) {
	dsn := DSN(
		item.GetString("host"),
		item.GetInt("port"),
		item.GetString("dbname"),
		item.GetString("username"),
		item.GetString("password"),
		item.GetString("sslmode"),
	)

	metrics, collectErr := CollectPostgres(context.Background(), dsn)
	now := time.Now()
	message := ""
	if collectErr != nil {
		message = collectErr.Error()
		metrics = Metrics{Up: false}
	}

	inventoryCol, err := app.FindCollectionByNameOrId("db_inventory")
	if err != nil {
		return
	}
	if item.GetString("version") == "" || metrics.Up {
		item.Set("version", metrics.Version)
	}
	if metrics.Up {
		item.Set("last_ok", now)
		item.Set("last_error", "")
	} else {
		item.Set("last_error", truncate(message, 480))
	}
	_ = app.Save(item)

	metricsCol, err := app.FindCollectionByNameOrId("db_metrics")
	if err != nil {
		_ = inventoryCol
		return
	}
	record := core.NewRecord(metricsCol)
	record.Set("inventory", item.Id)
	record.Set("collected", now)
	record.Set("up", metrics.Up)
	record.Set("version", metrics.Version)
	record.Set("uptime_sec", metrics.UptimeSec)
	record.Set("connections_total", metrics.ConnsTotal)
	record.Set("connections_active", metrics.ConnsActive)
	record.Set("connections_idle", metrics.ConnsIdle)
	record.Set("connections_max", metrics.ConnsMax)
	record.Set("connection_usage", round2(metrics.ConnUsage))
	record.Set("long_queries", metrics.LongQueries)
	record.Set("blocked_sessions", metrics.Blocked)
	record.Set("lock_wait_sec", round2(metrics.LockWaitSec))
	record.Set("replication_lag_sec", round2(metrics.ReplicationLag))
	record.Set("db_size_bytes", metrics.DBSizeBytes)
	record.Set("oldest_tx_sec", round2(metrics.OldestTxSec))
	record.Set("details", map[string]any{"error": message})
	if err := app.Save(record); err != nil {
		app.Logger().Error("dbobs: failed to save metrics", "err", err, "db", item.GetString("name"))
	}

	evaluateAlerts(app, item, metrics, rules, notify)
	pruneMetrics(app, item.Id)
}

type rule struct {
	Warning  float64
	Critical float64
	Enabled  bool
}

func loadRules(app core.App) map[string]rule {
	rules := map[string]rule{}
	records, err := app.FindAllRecords("db_alert_rules")
	if err == nil && len(records) > 0 {
		for _, r := range records {
			rules[r.GetString("metric")] = rule{
				Warning:  r.GetFloat("warning"),
				Critical: r.GetFloat("critical"),
				Enabled:  r.GetBool("enabled"),
			}
		}
		return rules
	}
	for _, d := range DefaultRules {
		rules[d.Metric] = rule{Warning: d.Warning, Critical: d.Critical, Enabled: true}
	}
	return rules
}

func evaluateAlerts(app core.App, item *core.Record, metrics Metrics, rules map[string]rule, notify Notifier) {
	values := map[string]float64{
		"connection_usage":    metrics.ConnUsage,
		"long_queries":        float64(metrics.LongQueries),
		"blocked_sessions":    float64(metrics.Blocked),
		"lock_wait_sec":       metrics.LockWaitSec,
		"replication_lag_sec": metrics.ReplicationLag,
		"oldest_tx_sec":       metrics.OldestTxSec,
	}
	if !metrics.Up {
		values["down"] = 1
	}

	for metric, value := range values {
		ruleCfg, ok := rules[metric]
		if !ok || !ruleCfg.Enabled {
			continue
		}
		severity := ""
		switch {
		case metric == "down":
			severity = "critical"
		case value >= ruleCfg.Critical:
			severity = "critical"
		case value >= ruleCfg.Warning:
			severity = "warning"
		}
		applyAlert(app, item, metric, severity, value, notify)
	}
}

func applyAlert(app core.App, item *core.Record, metric, severity string, value float64, notify Notifier) {
	open, err := app.FindFirstRecordByFilter(
		"db_alerts",
		"inventory={:inv} && metric={:m} && status != 'resolved'",
		dbx.Params{"inv": item.Id, "m": metric},
	)
	hasOpen := err == nil && open != nil

	if severity == "" {
		if hasOpen {
			open.Set("status", "resolved")
			open.Set("resolved", time.Now())
			open.Set("value", round2(value))
			if app.Save(open) == nil {
				label := item.GetString("name")
				notify(fmt.Sprintf("[RESOLVED] %s / %s", label, metric),
					fmt.Sprintf("Database: %s\nMetric: %s\nTime: %s WIB", label, metric, time.Now().Format("2006-01-02 15:04")))
			}
		}
		return
	}

	label := item.GetString("name")
	text := fmt.Sprintf("Database: %s\nMetric: %s\nValue: %.2f\nTime: %s WIB",
		label, metric, value, time.Now().Format("2006-01-02 15:04"))

	if hasOpen {
		previous := open.GetString("status")
		open.Set("severity", severity)
		open.Set("value", round2(value))
		open.Set("message", text)
		if previous == "acknowledged" {
			open.Set("status", "acknowledged")
		} else {
			open.Set("status", severity)
		}
		_ = app.Save(open)
		return
	}

	alertsCol, err := app.FindCollectionByNameOrId("db_alerts")
	if err != nil {
		return
	}
	record := core.NewRecord(alertsCol)
	record.Set("inventory", item.Id)
	record.Set("metric", metric)
	record.Set("severity", severity)
	record.Set("status", severity)
	record.Set("value", round2(value))
	record.Set("message", text)
	record.Set("triggered", time.Now())
	if err := app.Save(record); err != nil {
		return
	}
	notify(fmt.Sprintf("[%s] %s / %s", upper(severity), label, metric), text)
}

// pruneMetrics keeps raw samples for 30 days, matching the retention design.
func pruneMetrics(app core.App, inventoryID string) {
	cutoff := time.Now().AddDate(0, 0, -30)
	_, err := app.DB().
		NewQuery("DELETE FROM db_metrics WHERE inventory = {:inv} AND collected < {:cutoff}").
		Bind(dbx.Params{"inv": inventoryID, "cutoff": cutoff}).
		Execute()
	if err != nil {
		app.Logger().Debug("dbobs: metric retention cleanup failed", "err", err)
	}
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func round2(value float64) float64 {
	return float64(int64(value*100+0.5)) / 100
}

func upper(value string) string {
	if value == "" {
		return value
	}
	return string(value[0]-32) + value[1:]
}
