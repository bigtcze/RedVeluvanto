package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		col := core.NewBaseCollection("reddit_accounts", "col_reddit_accounts")

		col.Fields.Add(&core.RelationField{Name: "user", CollectionId: "_pb_users_auth_", MaxSelect: 1, Required: true})
		col.Fields.Add(&core.TextField{Name: "reddit_username", Max: 255})
		col.Fields.Add(&core.TextField{Name: "access_token", Max: 2000})
		col.Fields.Add(&core.TextField{Name: "refresh_token", Max: 2000})
		col.Fields.Add(&core.DateField{Name: "token_expiry"})
		col.Fields.Add(&core.JSONField{Name: "scopes", MaxSize: 1048576})
		col.Fields.Add(&core.BoolField{Name: "is_active"})
		col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		col.ListRule = types.Pointer("user = @request.auth.id")
		col.ViewRule = types.Pointer("user = @request.auth.id")
		col.CreateRule = nil
		col.UpdateRule = nil
		col.DeleteRule = nil

		col.AddIndex("idx_reddit_accounts_user_unique", true, "user", "")

		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("reddit_accounts")
		if err != nil {
			return err
		}
		return app.Delete(col)
	})
}
