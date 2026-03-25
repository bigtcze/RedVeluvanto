package ai

import "github.com/pocketbase/pocketbase/core"

func LoadProductContext(app core.App) *ProductContext {
	pc, err := app.FindFirstRecordByFilter("product_context", "id != ''")
	if err != nil {
		return nil
	}
	return &ProductContext{
		Name:            pc.GetString("name"),
		Description:     pc.GetString("description"),
		TargetAudience:  pc.GetString("target_audience"),
		KeyFeatures:     pc.GetString("key_features"),
		Differentiators: pc.GetString("differentiators"),
	}
}
