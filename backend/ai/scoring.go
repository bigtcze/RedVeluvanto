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
	var systemContent string
	if product != nil && product.Name != "" {
		systemContent = fmt.Sprintf(`You are a relevance scoring engine. Evaluate how relevant this Reddit thread is as an opportunity for the product "%s".

Consider both keyword relevance and whether the thread presents a genuine opportunity to engage (e.g., someone asking for recommendations, discussing a problem the product solves, comparing alternatives).

Score 0-100 where:
- 90-100: OP is actively looking for a solution the product offers, high-engagement thread, perfect opportunity
- 70-89: Thread is highly relevant to the product's domain, natural opportunity to contribute
- 40-69: Thread is somewhat related, product could be mentioned but it would feel forced
- 10-39: Loose connection, keyword match but not a real opportunity
- 0-9: False positive, completely irrelevant context

Respond ONLY with valid JSON: {"score": <number>, "reason": "<brief explanation>"}`, product.Name)
	} else {
		systemContent = `You are a relevance scoring engine. Evaluate how relevant this Reddit thread is for someone monitoring the keyword.

Score 0-100 where:
- 90-100: Thread directly discusses the exact topic, active discussion, perfect opportunity to engage
- 70-89: Thread is highly relevant, topic is discussed but not the main focus
- 40-69: Thread is somewhat related, keyword appears in context but tangentially
- 10-39: Thread has loose connection to the keyword
- 0-9: False positive, keyword match but completely irrelevant context

Respond ONLY with valid JSON: {"score": <number>, "reason": "<brief explanation>"}`
	}

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
