package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

var StyleInterviewQuestions = []string{
	"Someone asks for a tool recommendation in your product's category. How would you reply? (2-3 sentences, as you'd write on Reddit)",
	"A colleague on Slack suggests a solution you disagree with. What do you write back?",
	"A customer complains about a bug in your product. How do you respond?",
	"Write a short post about a product you genuinely like and use.",
	"Someone comments 'this is overrated, [competitor] is way better'. How do you reply?",
}

type StyleInterviewResult struct {
	Examples []string `json:"examples"`
}

func (c *Client) AnalyzeStyleAndGenerateExamples(ctx context.Context, answers []string) (*StyleInterviewResult, error) {
	var answersBlock strings.Builder
	for i, a := range answers {
		if i < len(StyleInterviewQuestions) {
			answersBlock.WriteString(fmt.Sprintf("Q: %s\nA: %s\n\n", StyleInterviewQuestions[i], a))
		}
	}

	systemMsg := Message{
		Role: "system",
		Content: `You are a writing style analyst. You will receive 5 answers from a person showing how they naturally write online.

Analyze their writing style:
- Sentence length and structure
- Level of formality/casualness
- Use of humor, sarcasm, or directness
- How they handle disagreement
- Typical openings and closings
- Use of contractions, slang, or abbreviations
- Whether they use emoji, exclamation marks, etc.

Then generate exactly 4 example Reddit comments written in their exact style. Each example should be a realistic reply to a different type of Reddit thread (asking for recommendations, sharing an experience, disagreeing with someone, explaining something).

The examples must sound like the SAME PERSON wrote them — match their voice, not just their topics.

Respond ONLY with valid JSON:
{
  "examples": ["example reply 1", "example reply 2", "example reply 3", "example reply 4"]
}`,
	}

	userMsg := Message{
		Role:    "user",
		Content: answersBlock.String(),
	}

	content, err := c.ChatCompletion(ctx, []Message{systemMsg, userMsg}, 0.3)
	if err != nil {
		return nil, fmt.Errorf("style analysis: %w", err)
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

	var result StyleInterviewResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parsing style result %q: %w", content, err)
	}

	return &result, nil
}
