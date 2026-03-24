package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		col := core.NewBaseCollection("personas", "col_personas")

		col.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 255})
		col.Fields.Add(&core.TextField{Name: "description", Max: 2000})
		col.Fields.Add(&core.JSONField{Name: "traits", MaxSize: 1048576})
		col.Fields.Add(&core.JSONField{Name: "custom_traits", MaxSize: 1048576})
		col.Fields.Add(&core.SelectField{Name: "reply_goal", Values: []string{"help", "promote", "reputation", "traffic", "educate"}, MaxSelect: 1})
		col.Fields.Add(&core.TextField{Name: "reply_goal_detail", Max: 2000})
		col.Fields.Add(&core.JSONField{Name: "behavior_rules", MaxSize: 1048576})
		col.Fields.Add(&core.SelectField{Name: "competitor_stance", Values: []string{"ignore", "acknowledge", "compare_fairly", "differentiate"}, MaxSelect: 1})
		col.Fields.Add(&core.JSONField{Name: "competitor_names", MaxSize: 1048576})
		col.Fields.Add(&core.JSONField{Name: "forbidden_words", MaxSize: 1048576})

		minVal := float64(0)
		col.Fields.Add(&core.NumberField{Name: "max_length", Min: &minVal, OnlyInt: true})
		col.Fields.Add(&core.TextField{Name: "language", Max: 10})
		col.Fields.Add(&core.TextField{Name: "knowledge_text", Max: 50000})
		col.Fields.Add(&core.JSONField{Name: "knowledge_urls", MaxSize: 1048576})
		col.Fields.Add(&core.JSONField{Name: "knowledge_cache", MaxSize: 1048576})
		col.Fields.Add(&core.JSONField{Name: "examples", MaxSize: 1048576})
		col.Fields.Add(&core.BoolField{Name: "is_default"})
		col.Fields.Add(&core.RelationField{Name: "created_by", CollectionId: "_pb_users_auth_", MaxSelect: 1, Required: true})
		col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		col.ListRule = types.Pointer("@request.auth.id != ''")
		col.ViewRule = types.Pointer("created_by = @request.auth.id")
		col.CreateRule = types.Pointer("@request.auth.id != ''")
		col.UpdateRule = types.Pointer("created_by = @request.auth.id")
		col.DeleteRule = types.Pointer("created_by = @request.auth.id")

		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("personas")
		if err != nil {
			return err
		}
		return app.Delete(col)
	})
}
