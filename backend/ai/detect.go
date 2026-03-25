package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type AIDetectionResult struct {
	Score int      `json:"score"`
	Flags []string `json:"flags"`
}

func (c *Client) DetectAIContent(ctx context.Context, text string) (*AIDetectionResult, error) {
	systemMsg := Message{
		Role: "system",
		Content: `Rate how much this Reddit comment sounds like it was written by AI versus a real person.

Score 0-10 where:
- 0-2: Clearly human — natural voice, personality, imperfections
- 3-5: Mostly human — minor hints of AI but passable
- 6-7: Suspicious — noticeable AI patterns (structured lists, filler phrases, generic tone)
- 8-10: Obviously AI — formulaic structure, buzzwords, no personality

Common AI tells to check for:
- Opens with a compliment ("Great question!", "Absolutely!")
- Uses "Furthermore", "Moreover", "Additionally", "It's worth noting"
- Ends with "Hope this helps!", "Let me know if you have questions!"
- Overuses superlatives ("incredibly", "game-changer", "robust")
- Perfectly structured with bullet points and bold headers
- Generic advice that could apply to anything
- No personal experience or opinion, just restated facts

Respond ONLY with valid JSON: {"score": <number>, "flags": ["specific issue 1", "specific issue 2"]}
Keep flags short (max 5 words each). Return empty flags array if score <= 5.`,
	}

	userMsg := Message{Role: "user", Content: text}

	content, err := c.ChatCompletion(ctx, []Message{systemMsg, userMsg}, 0)
	if err != nil {
		return nil, fmt.Errorf("ai detection: %w", err)
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

	var result AIDetectionResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return &AIDetectionResult{Score: 0}, nil
	}

	return &result, nil
}
