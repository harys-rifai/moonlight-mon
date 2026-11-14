package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds the remaining database engines to db_inventory.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("db_inventory")
		if err != nil {
			return err
		}
		field, ok := collection.Fields.GetByName("engine").(*core.SelectField)
		if !ok || field == nil {
			return nil
		}
		have := map[string]bool{}
		for _, value := range field.Values {
			have[value] = true
		}
		for _, value := range []string{"mariadb", "redis", "db2"} {
			if !have[value] {
				field.Values = append(field.Values, value)
			}
		}
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("db_inventory")
		if err != nil {
			return err
		}
		field, ok := collection.Fields.GetByName("engine").(*core.SelectField)
		if !ok || field == nil {
			return nil
		}
		keep := make([]string, 0, len(field.Values))
		for _, value := range field.Values {
			if value != "mariadb" && value != "redis" && value != "db2" {
				keep = append(keep, value)
			}
		}
		field.Values = keep
		return app.Save(collection)
	})
}
