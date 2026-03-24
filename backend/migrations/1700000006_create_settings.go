package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		col := core.NewBaseCollection("settings", "col_settings")

		col.Fields.Add(&core.TextField{Name: "key", Required: true, Max: 255})
		col.Fields.Add(&core.JSONField{Name: "value", MaxSize: 1048576})
		col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		col.ListRule = types.Pointer("@request.auth.id != ''")
		col.ViewRule = types.Pointer("@request.auth.id != ''")
		col.CreateRule = nil
		col.UpdateRule = nil
		col.DeleteRule = nil

		col.AddIndex("idx_settings_key_unique", true, "key", "")

		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		return app.Delete(col)
	})
}
