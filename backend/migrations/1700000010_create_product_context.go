package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		col := core.NewBaseCollection("product_context", "col_product_context")

		col.Fields.Add(&core.TextField{Name: "name", Max: 255})
		col.Fields.Add(&core.TextField{Name: "description", Max: 5000})
		col.Fields.Add(&core.TextField{Name: "target_audience", Max: 2000})
		col.Fields.Add(&core.TextField{Name: "key_features", Max: 5000})
		col.Fields.Add(&core.TextField{Name: "differentiators", Max: 5000})
		col.Fields.Add(&core.TextField{Name: "website_url", Max: 500})
		col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		col.ListRule = types.Pointer("@request.auth.id != ''")
		col.ViewRule = types.Pointer("@request.auth.id != ''")
		col.CreateRule = nil
		col.UpdateRule = nil
		col.DeleteRule = nil

		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("product_context")
		if err != nil {
			return err
		}
		return app.Delete(col)
	})
}
