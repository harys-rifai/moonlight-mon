package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Database observability collections: inventory, collected metrics, alert rules, and alert lifecycle.
func init() {
	m.Register(func(app core.App) error {
		readRule := types.Pointer(`@request.auth.id != ""`)
		writeRule := types.Pointer(`@request.auth.id != "" && @request.auth.role = "admin"`)

		inventory := core.NewBaseCollection("db_inventory")
		inventory.ListRule, inventory.ViewRule = readRule, readRule
		inventory.CreateRule, inventory.UpdateRule, inventory.DeleteRule = writeRule, writeRule, writeRule
		inventory.Fields.Add(
			&core.TextField{Name: "name", Required: true, Max: 120},
			&core.TextField{Name: "application", Max: 120},
			&core.SelectField{Name: "environment", Values: []string{"PROD", "STAGING", "UAT", "DEV", "DR", "OTHER"}, MaxSelect: 1},
			&core.SelectField{Name: "engine", Values: []string{"postgres", "mysql", "sqlserver", "oracle"}, MaxSelect: 1},
			&core.TextField{Name: "host", Required: true, Max: 255},
			&core.NumberField{Name: "port", Min: types.Pointer(0.0), Max: types.Pointer(65535.0), OnlyInt: true},
			&core.TextField{Name: "dbname", Max: 120},
			&core.TextField{Name: "username", Max: 120},
			&core.TextField{Name: "password", Max: 255, Hidden: true},
			&core.SelectField{Name: "sslmode", Values: []string{"disable", "require", "verify-full"}, MaxSelect: 1},
			&core.TextField{Name: "owner", Max: 120},
			&core.TextField{Name: "version", Max: 120},
			&core.TextField{Name: "ha_mode", Max: 120},
			&core.TextField{Name: "cloud_provider", Max: 120},
			&core.TextField{Name: "group_name", Max: 120},
			&core.TextField{Name: "notes", Max: 1000},
			&core.BoolField{Name: "enabled"},
			&core.DateField{Name: "last_ok"},
			&core.TextField{Name: "last_error", Max: 500},
		)
		inventory.AddIndex("idx_db_inventory_name", true, "name", "")
		if err := app.Save(inventory); err != nil {
			return err
		}

		metrics := core.NewBaseCollection("db_metrics")
		metrics.ListRule, metrics.ViewRule = readRule, readRule
		metrics.CreateRule, metrics.UpdateRule, metrics.DeleteRule = writeRule, writeRule, writeRule
		metrics.Fields.Add(
			&core.RelationField{Name: "inventory", CollectionId: inventory.Id, Required: true, CascadeDelete: true, MaxSelect: 1},
			&core.DateField{Name: "collected", Required: true},
			&core.BoolField{Name: "up"},
			&core.TextField{Name: "version", Max: 120},
			&core.NumberField{Name: "uptime_sec"},
			&core.NumberField{Name: "connections_total"},
			&core.NumberField{Name: "connections_active"},
			&core.NumberField{Name: "connections_idle"},
			&core.NumberField{Name: "connections_max"},
			&core.NumberField{Name: "connection_usage"},
			&core.NumberField{Name: "long_queries"},
			&core.NumberField{Name: "blocked_sessions"},
			&core.NumberField{Name: "lock_wait_sec"},
			&core.NumberField{Name: "replication_lag_sec"},
			&core.NumberField{Name: "db_size_bytes"},
			&core.NumberField{Name: "oldest_tx_sec"},
			&core.JSONField{Name: "details", MaxSize: 20000},
		)
		metrics.AddIndex("idx_db_metrics_inventory_collected", false, "inventory, collected", "")
		if err := app.Save(metrics); err != nil {
			return err
		}

		rulesCol := core.NewBaseCollection("db_alert_rules")
		rulesCol.ListRule, rulesCol.ViewRule = readRule, readRule
		rulesCol.CreateRule, rulesCol.UpdateRule, rulesCol.DeleteRule = writeRule, writeRule, writeRule
		rulesCol.Fields.Add(
			&core.SelectField{Name: "metric", Required: true, MaxSelect: 1, Values: []string{
				"down", "connection_usage", "long_queries", "blocked_sessions",
				"lock_wait_sec", "replication_lag_sec", "oldest_tx_sec",
			}},
			&core.NumberField{Name: "warning"},
			&core.NumberField{Name: "critical"},
			&core.BoolField{Name: "enabled"},
		)
		rulesCol.AddIndex("idx_db_alert_rules_metric", true, "metric", "")
		if err := app.Save(rulesCol); err != nil {
			return err
		}

		alertsCol := core.NewBaseCollection("db_alerts")
		alertsCol.ListRule, alertsCol.ViewRule = readRule, readRule
		alertsCol.CreateRule, alertsCol.UpdateRule, alertsCol.DeleteRule = writeRule, writeRule, writeRule
		alertsCol.Fields.Add(
			&core.RelationField{Name: "inventory", CollectionId: inventory.Id, Required: true, CascadeDelete: true, MaxSelect: 1},
			&core.TextField{Name: "metric", Required: true, Max: 60},
			&core.SelectField{Name: "severity", MaxSelect: 1, Values: []string{"warning", "critical"}},
			&core.SelectField{Name: "status", MaxSelect: 1, Values: []string{"new", "warning", "critical", "acknowledged", "resolved"}},
			&core.NumberField{Name: "value"},
			&core.TextField{Name: "message", Max: 1000},
			&core.DateField{Name: "triggered"},
			&core.DateField{Name: "acknowledged"},
			&core.DateField{Name: "resolved"},
		)
		alertsCol.AddIndex("idx_db_alerts_inventory_metric", false, "inventory, metric", "")
		return app.Save(alertsCol)
	}, func(app core.App) error {
		for _, name := range []string{"db_alerts", "db_alert_rules", "db_metrics", "db_inventory"} {
			collection, err := app.FindCollectionByNameOrId(name)
			if err == nil {
				if err := app.Delete(collection); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
