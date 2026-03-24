package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		col := core.NewBaseCollection("drafts", "col_drafts")

		col.Fields.Add(&core.RelationField{Name: "thread", CollectionId: "col_threads", MaxSelect: 1, Required: true})
		col.Fields.Add(&core.RelationField{Name: "persona", CollectionId: "col_personas", MaxSelect: 1})
		col.Fields.Add(&core.RelationField{Name: "user", CollectionId: "_pb_users_auth_", MaxSelect: 1, Required: true})
		col.Fields.Add(&core.TextField{Name: "parent_comment_id", Max: 20})
		col.Fields.Add(&core.TextField{Name: "generated_text", Max: 50000, Required: true})
		col.Fields.Add(&core.TextField{Name: "edited_text", Max: 50000})
		col.Fields.Add(&core.SelectField{Name: "status", Values: []string{"draft", "approved", "posted", "failed"}, MaxSelect: 1, Required: true})
		col.Fields.Add(&core.DateField{Name: "posted_at"})
		col.Fields.Add(&core.TextField{Name: "reddit_comment_id", Max: 20})
		col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		col.ListRule = types.Pointer("user = @request.auth.id")
		col.ViewRule = types.Pointer("user = @request.auth.id")
		col.CreateRule = types.Pointer("@request.auth.id != ''")
		col.UpdateRule = types.Pointer("user = @request.auth.id")
		col.DeleteRule = types.Pointer("user = @request.auth.id")

		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("drafts")
		if err != nil {
			return err
		}
		return app.Delete(col)
	})
}
