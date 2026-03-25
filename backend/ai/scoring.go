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

type ProductContext struct {
	Name            string
	Description     string
	TargetAudience  string
	KeyFeatures     string
	Differentiators string
}

func (c *Client) ScoreThread(ctx context.Context, title, body, subreddit, subredditDescription, keyword string, product *ProductContext) (*ScoringResult, error) {
	context := "someone monitoring the keyword"
	if product != nil && product.Name != "" {
		context = fmt.Sprintf("the product \"%s\"", product.Name)
	}

	systemContent := fmt.Sprintf(`Relevance scoring engine. Score how relevant this Reddit thread is for %s.

Score 0-100:
- 90-100: Direct match, OP seeking solutions, perfect engagement opportunity
- 70-89: Highly relevant, natural opportunity to contribute
- 40-69: Somewhat related, mention would feel forced
- 10-39: Loose connection, not a real opportunity
- 0-9: False positive, irrelevant context

Respond ONLY with valid JSON: {"score": <number>, "reason": "<brief explanation>"}`, context)

	systemMsg := Message{Role: "system", Content: systemContent}

	var userParts []string
	if product != nil && product.Name != "" {
		productSection := fmt.Sprintf("=== YOUR PRODUCT ===\n%s — %s", product.Name, product.Description)
		if product.TargetAudience != "" {
			productSection += fmt.Sprintf("\nTarget audience: %s", product.TargetAudience)
		}
		if product.Differentiators != "" {
			productSection += fmt.Sprintf("\nDifferentiators: %s", product.Differentiators)
		}
		userParts = append(userParts, productSection)
	}
	userParts = append(userParts, fmt.Sprintf(
		"=== THREAD ===\nKeyword: %s\nSubreddit: r/%s\nSubreddit description: %s\nThread title: %s\nThread body: %s",
		keyword, subreddit, subredditDescription, title, body,
	))

	userMsg := Message{Role: "user", Content: strings.Join(userParts, "\n\n")}

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
