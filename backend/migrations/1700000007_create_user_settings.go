package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		col := core.NewBaseCollection("user_settings", "col_user_settings")

		col.Fields.Add(&core.RelationField{Name: "user", CollectionId: "_pb_users_auth_", MaxSelect: 1, Required: true})
		col.Fields.Add(&core.TextField{Name: "discord_webhook_url", Max: 500})
		col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		col.ListRule = types.Pointer("user = @request.auth.id")
		col.ViewRule = types.Pointer("user = @request.auth.id")
		col.CreateRule = types.Pointer("@request.auth.id != ''")
		col.UpdateRule = types.Pointer("user = @request.auth.id")
		col.DeleteRule = nil

		col.AddIndex("idx_user_settings_user_unique", true, "user", "")

		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("user_settings")
		if err != nil {
			return err
		}
		return app.Delete(col)
	})
}
