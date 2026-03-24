package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		col := core.NewBaseCollection("keywords", "col_keywords")

		col.Fields.Add(&core.TextField{Name: "keyword", Required: true, Max: 255})
		col.Fields.Add(&core.JSONField{Name: "subreddits", MaxSize: 1048576})
		col.Fields.Add(&core.BoolField{Name: "is_active"})
		col.Fields.Add(&core.RelationField{Name: "created_by", CollectionId: "_pb_users_auth_", MaxSelect: 1, Required: true})
		col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		col.ListRule = types.Pointer("@request.auth.id != ''")
		col.ViewRule = types.Pointer("@request.auth.id != ''")
		col.CreateRule = types.Pointer("@request.auth.id != ''")
		col.UpdateRule = types.Pointer("created_by = @request.auth.id")
		col.DeleteRule = types.Pointer("created_by = @request.auth.id")

		col.AddIndex("idx_keywords_keyword_unique", true, "keyword", "")

		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("keywords")
		if err != nil {
			return err
		}
		return app.Delete(col)
	})
}
