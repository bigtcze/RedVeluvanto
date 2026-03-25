package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("drafts")
		if err != nil {
			return err
		}

		minScore := float64(0)
		maxScore := float64(10)
		col.Fields.Add(&core.NumberField{Name: "ai_detection_score", Min: &minScore, Max: &maxScore, OnlyInt: true})
		col.Fields.Add(&core.JSONField{Name: "ai_detection_flags", MaxSize: 10000})

		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("drafts")
		if err != nil {
			return err
		}

		col.Fields.RemoveByName("ai_detection_score")
		col.Fields.RemoveByName("ai_detection_flags")

		return app.Save(col)
	})
}
