package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Adds deep PostgreSQL metrics (TPS, slow queries, locks, cache/temp/WAL, APM samples).
func init() {
	m.Register(func(app core.App) error {
		metrics, err := app.FindCollectionByNameOrId("db_metrics")
		if err != nil {
			return err
		}
		existing := map[string]bool{}
		for _, field := range metrics.Fields {
			existing[field.GetName()] = true
		}
		add := func(name string) {
			if !existing[name] {
				metrics.Fields.Add(&core.NumberField{Name: name})
			}
		}
		for _, name := range []string{
			"tps", "qps", "queries_total", "xact_commit", "xact_rollback",
			"deadlocks", "deadlocks_rate", "cache_hit_ratio", "blks_read", "blks_hit",
			"temp_bytes", "temp_bytes_rate", "wal_bytes", "wal_bytes_rate",
			"slow_queries", "idle_in_tx", "waiting_locks", "blk_read_time_ms", "blk_write_time_ms",
		} {
			add(name)
		}
		if err := app.Save(metrics); err != nil {
			return err
		}

		rules, err := app.FindCollectionByNameOrId("db_alert_rules")
		if err == nil {
			if field, ok := rules.Fields.GetByName("metric").(*core.SelectField); ok {
				for _, value := range []string{"slow_queries", "deadlocks_rate", "idle_in_tx", "waiting_locks"} {
					found := false
					for _, v := range field.Values {
						if v == value {
							found = true
						}
					}
					if !found {
						field.Values = append(field.Values, value)
					}
				}
			}
			if err := app.Save(rules); err != nil {
				return err
			}
		}

		samples := core.NewBaseCollection("db_query_samples")
		inventory, err := app.FindCollectionByNameOrId("db_inventory")
		if err != nil {
			return err
		}
		readRule := types.Pointer(`@request.auth.id != ""`)
		writeRule := types.Pointer(`@request.auth.id != "" && @request.auth.role = "admin"`)
		samples.ListRule, samples.ViewRule = readRule, readRule
		samples.CreateRule, samples.UpdateRule, samples.DeleteRule = writeRule, writeRule, writeRule
		samples.Fields.Add(
			&core.RelationField{Name: "inventory", CollectionId: inventory.Id, Required: true, MaxSelect: 1},
			&core.DateField{Name: "collected", Required: true},
			&core.TextField{Name: "query", Max: 2000},
			&core.TextField{Name: "role", Max: 120},
			&core.NumberField{Name: "calls"},
			&core.NumberField{Name: "mean_ms"},
			&core.NumberField{Name: "total_ms"},
			&core.NumberField{Name: "rows"},
			&core.NumberField{Name: "cache_hit_ratio"},
		)
		samples.AddIndex("idx_db_query_samples_inventory_collected", false, "inventory, collected", "")
		return app.Save(samples)
	}, func(app core.App) error {
		if samples, err := app.FindCollectionByNameOrId("db_query_samples"); err == nil {
			if err := app.Delete(samples); err != nil {
				return err
			}
		}
		return nil
	})
}
