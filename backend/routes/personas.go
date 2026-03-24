package routes

import (
	"encoding/json"
	"net/http"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"redveluvanto/ai"
)

func RegisterPersonaRoutes(e *core.ServeEvent, aiClient *ai.Client) {
	e.Router.POST("/api/personas/preview", func(re *core.RequestEvent) error {
		var body struct {
			Traits           map[string]float64 `json:"traits"`
			CustomTraits     []string           `json:"custom_traits"`
			ReplyGoal        string             `json:"reply_goal"`
			ReplyGoalDetail  string             `json:"reply_goal_detail"`
			BehaviorRules    []string           `json:"behavior_rules"`
			CompetitorStance string             `json:"competitor_stance"`
			CompetitorNames  []string           `json:"competitor_names"`
			ForbiddenWords   []string           `json:"forbidden_words"`
			MaxLength        int                `json:"max_length"`
			Language         string             `json:"language"`
			KnowledgeText    string             `json:"knowledge_text"`
			KnowledgeCache   string             `json:"knowledge_cache"`
			Examples         []string           `json:"examples"`
		}

		if err := json.NewDecoder(re.Request.Body).Decode(&body); err != nil {
			return re.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
		}

		systemPrompt := ai.BuildPersonaPrompt(
			body.Traits,
			body.CustomTraits,
			body.ReplyGoal,
			body.ReplyGoalDetail,
			body.BehaviorRules,
			body.CompetitorStance,
			body.CompetitorNames,
			body.ForbiddenWords,
			body.MaxLength,
			body.Language,
			body.KnowledgeText,
			body.KnowledgeCache,
			body.Examples,
		)

		userMessage := `=== SUBREDDIT RULES ===
1. Be respectful
2. No self-promotion spam

=== ORIGINAL POST ===
Title: Looking for a good tool to manage documents in our team
Author: u/curious_founder
Score: 42

We're a small startup (15 people) and our documents are everywhere - Google Docs, Notion, random folders. Looking for something that can bring order to this chaos. Any recommendations?

=== COMMENTS ===
u/tech_reviewer (score: 15):
  Have you looked at Notion? It's pretty good for small teams.

  └─ u/curious_founder (score: 8):
     Yeah we tried it but it gets messy fast with larger docs. Need something more structured.

=== YOUR TARGET ===
Reply to the comment by u/curious_founder: Yeah we tried it but it gets messy fast...

Generate your reply now.`

		messages := []ai.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		}

		preview, err := aiClient.ChatCompletion(re.Request.Context(), messages, 0.7)
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return re.JSON(http.StatusOK, map[string]string{
			"preview":        preview,
			"system_prompt": systemPrompt,
		})
	}).Bind(apis.RequireAuth())
}
