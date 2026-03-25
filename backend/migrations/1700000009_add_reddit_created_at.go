package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("threads")
		if err != nil {
			return err
		}

		col.Fields.Add(&core.DateField{Name: "reddit_created_at"})

		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("threads")
		if err != nil {
			return err
		}

		col.Fields.RemoveByName("reddit_created_at")

		return app.Save(col)
	})
}
