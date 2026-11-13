package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection := core.NewBaseCollection("test_collection")
		collection.Fields.Add(&core.TextField{Name: "name"})
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("test_collection")
		if err != nil {
			return err
		}
		return app.Delete(collection)
	})
}
