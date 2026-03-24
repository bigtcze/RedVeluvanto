package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

func NotifyUsers(app core.App, threadRecord *core.Record, keyword string) {
	settings, err := app.FindRecordsByFilter("user_settings", "discord_webhook_url != ''", "", 0, 0)
	if err != nil {
		log.Printf("worker: notify: failed to find user settings: %v", err)
		return
	}

	subreddit := threadRecord.GetString("subreddit")
	title := threadRecord.GetString("title")
	url := threadRecord.GetString("url")
	score := threadRecord.GetInt("relevance_score")

	payload := map[string]any{
		"embeds": []map[string]any{
			{
				"title":       fmt.Sprintf("Nový relevantní thread (skóre: %d)", score),
				"description": fmt.Sprintf("r/%s — \"%s\"", subreddit, title),
				"url":         url,
				"color":       5814783,
				"fields": []map[string]any{
					{"name": "Keyword", "value": keyword, "inline": true},
					{"name": "Relevance", "value": fmt.Sprintf("%d/100", score), "inline": true},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("worker: notify: failed to marshal payload: %v", err)
		return
	}

	for _, s := range settings {
		webhookURL := s.GetString("discord_webhook_url")
		resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
		if err != nil {
			log.Printf("worker: notify: webhook post failed: %v", err)
			continue
		}
		resp.Body.Close()
	}
}
