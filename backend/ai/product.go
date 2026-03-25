package ai

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

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

func LoadGlobalForbiddenWords(app core.App) []string {
	record, err := app.FindFirstRecordByFilter("settings", "key = {:k}", dbx.Params{"k": "global_forbidden_phrases"})
	if err != nil {
		return nil
	}
	var words []string
	if err := json.Unmarshal([]byte(record.GetString("value")), &words); err != nil {
		return nil
	}
	return words
}
