package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type ScoringResult struct {
	Score  int    `json:"score"`
	Reason string `json:"reason"`
}

func (c *Client) ScoreThread(ctx context.Context, title, body, subreddit, subredditDescription, keyword string) (*ScoringResult, error) {
	systemMsg := Message{
		Role: "system",
		Content: `You are a relevance scoring engine. Evaluate how relevant this Reddit thread is for someone monitoring the keyword.

Score 0-100 where:
- 90-100: Thread directly discusses the exact topic, active discussion, perfect opportunity to engage
- 70-89: Thread is highly relevant, topic is discussed but not the main focus
- 40-69: Thread is somewhat related, keyword appears in context but tangentially
- 10-39: Thread has loose connection to the keyword
- 0-9: False positive, keyword match but completely irrelevant context

Respond ONLY with valid JSON: {"score": <number>, "reason": "<brief explanation>"}`,
	}

	userMsg := Message{
		Role: "user",
		Content: fmt.Sprintf(
			"Keyword: %s\nSubreddit: r/%s\nSubreddit description: %s\nThread title: %s\nThread body: %s",
			keyword, subreddit, subredditDescription, title, body,
		),
	}

	content, err := c.ChatCompletion(ctx, []Message{systemMsg, userMsg}, 0)
	if err != nil {
		return nil, fmt.Errorf("chat completion: %w", err)
	}

	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		var jsonLines []string
		for _, line := range strings.Split(content, "\n") {
			if strings.HasPrefix(line, "```") {
				continue
			}
			jsonLines = append(jsonLines, line)
		}
		content = strings.Join(jsonLines, "\n")
	}

	var result ScoringResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parsing scoring result %q: %w", content, err)
	}

	return &result, nil
}
