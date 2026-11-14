package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("network_monitors")
		if err != nil {
			return err
		}
		field, ok := collection.Fields.GetByName("protocol").(*core.SelectField)
		if !ok || field == nil {
			return nil
		}
		for _, value := range field.Values {
			if value == "telnet" {
				return nil
			}
		}
		field.Values = append(field.Values, "telnet")
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("network_monitors")
		if err != nil {
			return err
		}
		field, ok := collection.Fields.GetByName("protocol").(*core.SelectField)
		if !ok || field == nil {
			return nil
		}
		values := make([]string, 0, len(field.Values))
		for _, value := range field.Values {
			if value != "telnet" {
				values = append(values, value)
			}
		}
		field.Values = values
		return app.Save(collection)
	})
}
