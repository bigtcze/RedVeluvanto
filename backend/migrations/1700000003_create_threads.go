package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		col := core.NewBaseCollection("threads", "col_threads")

		col.Fields.Add(&core.TextField{Name: "reddit_id", Required: true, Max: 20})
		col.Fields.Add(&core.TextField{Name: "subreddit", Required: true, Max: 255})
		col.Fields.Add(&core.TextField{Name: "title", Required: true, Max: 1000})
		col.Fields.Add(&core.TextField{Name: "body", Max: 50000})
		col.Fields.Add(&core.TextField{Name: "url", Max: 2000})
		col.Fields.Add(&core.TextField{Name: "author", Max: 255})

		col.Fields.Add(&core.NumberField{Name: "score", OnlyInt: true})
		col.Fields.Add(&core.NumberField{Name: "num_comments", OnlyInt: true})
		col.Fields.Add(&core.JSONField{Name: "comments_tree", MaxSize: 5242880})
		col.Fields.Add(&core.JSONField{Name: "subreddit_rules", MaxSize: 1048576})
		col.Fields.Add(&core.TextField{Name: "subreddit_description", Max: 10000})
		col.Fields.Add(&core.RelationField{Name: "matched_keyword", CollectionId: "col_keywords", MaxSelect: 1})

		minRelevance := float64(0)
		maxRelevance := float64(100)
		col.Fields.Add(&core.NumberField{Name: "relevance_score", Min: &minRelevance, Max: &maxRelevance, OnlyInt: true})
		col.Fields.Add(&core.TextField{Name: "relevance_reason", Max: 2000})
		col.Fields.Add(&core.DateField{Name: "found_at"})
		col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		col.ListRule = types.Pointer("@request.auth.id != ''")
		col.ViewRule = types.Pointer("@request.auth.id != ''")
		col.CreateRule = nil
		col.UpdateRule = nil
		col.DeleteRule = nil

		col.AddIndex("idx_threads_reddit_id_unique", true, "reddit_id", "")

		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("threads")
		if err != nil {
			return err
		}
		return app.Delete(col)
	})
}
